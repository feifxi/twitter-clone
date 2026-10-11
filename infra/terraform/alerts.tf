# AWS-side alerting, all by email: is the site reachable from outside, is the instance
# healthy (and recovered automatically if not), is the database running out of room.
# Application alerts (error rate, latency, disk, …) live in Grafana: see ../grafana.

resource "aws_sns_topic" "alerts" {
  name = "${local.name}-alerts"
}

resource "aws_sns_topic_subscription" "alerts_email" {
  topic_arn = aws_sns_topic.alerts.arn
  protocol  = "email"
  endpoint  = var.alert_email
}

# ── Uptime, checked from outside AWS's view of the instance ──

resource "aws_route53_health_check" "site" {
  fqdn              = var.domain
  type              = "HTTPS"
  port              = 443
  resource_path     = "/readyz" # API up and database reachable, through nginx
  request_interval  = 30
  failure_threshold = 3
  tags              = { Name = "${local.name}-site" }
}

resource "aws_sns_topic" "alerts_us_east_1" {
  provider = aws.us_east_1
  name     = "${local.name}-alerts"
}

resource "aws_sns_topic_subscription" "alerts_email_us_east_1" {
  provider  = aws.us_east_1
  topic_arn = aws_sns_topic.alerts_us_east_1.arn
  protocol  = "email"
  endpoint  = var.alert_email
}

resource "aws_cloudwatch_metric_alarm" "site_down" {
  provider            = aws.us_east_1
  alarm_name          = "${local.name}-site-down"
  alarm_description   = "https://${var.domain}/readyz has been failing for 2 minutes"
  namespace           = "AWS/Route53"
  metric_name         = "HealthCheckStatus"
  dimensions          = { HealthCheckId = aws_route53_health_check.site.id }
  statistic           = "Minimum"
  period              = 60
  evaluation_periods  = 2
  comparison_operator = "LessThanThreshold"
  threshold           = 1
  treat_missing_data  = "breaching"
  alarm_actions       = [aws_sns_topic.alerts_us_east_1.arn]
  ok_actions          = [aws_sns_topic.alerts_us_east_1.arn]
}

# ── Instance ────────────────────────────────────────

# The AWS host underneath failed: move the instance to healthy hardware (same IP, disk).
resource "aws_cloudwatch_metric_alarm" "instance_system_check" {
  alarm_name          = "${local.name}-instance-system-check"
  alarm_description   = "AWS hardware/network problem under the instance: recovering it automatically"
  namespace           = "AWS/EC2"
  metric_name         = "StatusCheckFailed_System"
  dimensions          = { InstanceId = aws_instance.app.id }
  statistic           = "Maximum"
  period              = 60
  evaluation_periods  = 2
  comparison_operator = "GreaterThanOrEqualToThreshold"
  threshold           = 1
  alarm_actions       = ["arn:aws:automate:${var.aws_region}:ec2:recover", aws_sns_topic.alerts.arn]
  ok_actions          = [aws_sns_topic.alerts.arn]
}

# The OS stopped responding (kernel panic, out of memory): reboot it.
resource "aws_cloudwatch_metric_alarm" "instance_check" {
  alarm_name          = "${local.name}-instance-check"
  alarm_description   = "The instance's OS isn't responding: rebooting it automatically"
  namespace           = "AWS/EC2"
  metric_name         = "StatusCheckFailed_Instance"
  dimensions          = { InstanceId = aws_instance.app.id }
  statistic           = "Maximum"
  period              = 60
  evaluation_periods  = 3
  comparison_operator = "GreaterThanOrEqualToThreshold"
  threshold           = 1
  alarm_actions       = ["arn:aws:automate:${var.aws_region}:ec2:reboot", aws_sns_topic.alerts.arn]
  ok_actions          = [aws_sns_topic.alerts.arn]
}

# t3 instances run in "unlimited" mode: sustained high CPU costs extra, so say so.
resource "aws_cloudwatch_metric_alarm" "instance_cpu" {
  alarm_name          = "${local.name}-instance-cpu"
  alarm_description   = "Instance CPU above 80% for 30 minutes"
  namespace           = "AWS/EC2"
  metric_name         = "CPUUtilization"
  dimensions          = { InstanceId = aws_instance.app.id }
  statistic           = "Average"
  period              = 300
  evaluation_periods  = 6
  comparison_operator = "GreaterThanThreshold"
  threshold           = 80
  alarm_actions       = [aws_sns_topic.alerts.arn]
  ok_actions          = [aws_sns_topic.alerts.arn]
}

# ── Database ────────────────────────────────────────

locals {
  rds_alarms = {
    storage = {
      description = "Less than 2 GB of database storage left"
      metric      = "FreeStorageSpace"
      statistic   = "Minimum"
      operator    = "LessThanThreshold"
      threshold   = 2 * 1024 * 1024 * 1024
      periods     = 1
    }
    cpu = {
      description = "Database CPU above 80% for 15 minutes"
      metric      = "CPUUtilization"
      statistic   = "Average"
      operator    = "GreaterThanThreshold"
      threshold   = 80
      periods     = 3
    }
    memory = {
      description = "Less than 64 MB of free database memory for 15 minutes"
      metric      = "FreeableMemory"
      statistic   = "Average"
      operator    = "LessThanThreshold"
      threshold   = 64 * 1024 * 1024
      periods     = 3
    }
  }
}

resource "aws_cloudwatch_metric_alarm" "rds" {
  for_each            = local.rds_alarms
  alarm_name          = "${local.name}-db-${each.key}"
  alarm_description   = each.value.description
  namespace           = "AWS/RDS"
  metric_name         = each.value.metric
  dimensions          = { DBInstanceIdentifier = aws_db_instance.main.identifier }
  statistic           = each.value.statistic
  period              = 300
  evaluation_periods  = each.value.periods
  comparison_operator = each.value.operator
  threshold           = each.value.threshold
  alarm_actions       = [aws_sns_topic.alerts.arn]
  ok_actions          = [aws_sns_topic.alerts.arn]
}
