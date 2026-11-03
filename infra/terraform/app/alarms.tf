# AWS-side safety nets, by email. They cover what the monitoring stack on the host can't see
# about itself (the host being down) and the bill. App alerts (SLOs, errors, disk) come from
# Prometheus and Alertmanager: see infra/observability.

resource "aws_sns_topic" "alerts" {
  name = "${local.name}-alerts"
}

# AWS emails a confirmation link after each up; click it to receive the alarms
resource "aws_sns_topic_subscription" "email" {
  topic_arn = aws_sns_topic.alerts.arn
  protocol  = "email"
  endpoint  = var.alert_email
}

locals {
  instance_alarms = {
    # AWS hardware or network under the instance failed: move it to healthy hardware (same IP and disk)
    system-check = {
      metric = "StatusCheckFailed_System", stat = "Maximum", op = "GreaterThanOrEqualToThreshold", threshold = 1, periods = 2, period = 60
      action = "arn:aws:automate:${var.aws_region}:ec2:recover", desc = "AWS problem under the instance: recovering it automatically"
    }
    # The OS stopped responding (kernel panic, out of memory): reboot it
    instance-check = {
      metric = "StatusCheckFailed_Instance", stat = "Maximum", op = "GreaterThanOrEqualToThreshold", threshold = 1, periods = 3, period = 60
      action = "arn:aws:automate:${var.aws_region}:ec2:reboot", desc = "The instance's OS isn't responding: rebooting it automatically"
    }
    cpu = {
      metric = "CPUUtilization", stat = "Average", op = "GreaterThanThreshold", threshold = 80, periods = 3, period = 300
      action = null, desc = "Instance CPU above 80% for 15 minutes"
    }
  }
}

resource "aws_cloudwatch_metric_alarm" "instance" {
  for_each            = local.instance_alarms
  alarm_name          = "${local.name}-${each.key}"
  alarm_description   = each.value.desc
  namespace           = "AWS/EC2"
  metric_name         = each.value.metric
  dimensions          = { InstanceId = aws_instance.host.id }
  statistic           = each.value.stat
  period              = each.value.period
  evaluation_periods  = each.value.periods
  comparison_operator = each.value.op
  threshold           = each.value.threshold
  alarm_actions       = compact([each.value.action, aws_sns_topic.alerts.arn])
  ok_actions          = [aws_sns_topic.alerts.arn]
}

# Real spend, credits excluded, so the email means something while credits pay the bill
resource "aws_budgets_budget" "monthly" {
  name         = "${local.name}-monthly"
  budget_type  = "COST"
  limit_amount = tostring(var.monthly_budget_usd)
  limit_unit   = "USD"
  time_unit    = "MONTHLY"

  cost_types {
    include_credit = false
    include_refund = false
  }

  notification {
    comparison_operator        = "GREATER_THAN"
    threshold                  = 50
    threshold_type             = "PERCENTAGE"
    notification_type          = "ACTUAL"
    subscriber_email_addresses = [var.alert_email]
  }
  notification {
    comparison_operator        = "GREATER_THAN"
    threshold                  = 100
    threshold_type             = "PERCENTAGE"
    notification_type          = "FORECASTED"
    subscriber_email_addresses = [var.alert_email]
  }
}
