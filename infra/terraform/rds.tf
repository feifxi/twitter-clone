resource "aws_db_subnet_group" "main" {
  name       = "${local.name}-db"
  subnet_ids = aws_subnet.private[*].id
}

resource "aws_db_instance" "main" {
  identifier     = "${local.name}-db"
  engine         = "postgres"
  engine_version = "16"
  instance_class = var.rds_instance_class

  db_name  = var.db_name
  username = var.db_username
  password = var.db_password

  allocated_storage = 20
  storage_type      = "gp3"
  storage_encrypted = true # free; encrypts data, backups and snapshots at rest

  db_subnet_group_name   = aws_db_subnet_group.main.name
  vpc_security_group_ids = [aws_security_group.rds.id]
  publicly_accessible    = false

  # Single AZ keeps the cost down; backups and deletion protection keep the data safe.
  # To tear down on purpose: set deletion_protection = false and apply first.
  multi_az                   = false
  backup_retention_period    = 7 # point-in-time restore to any second of the last week
  deletion_protection        = true
  skip_final_snapshot        = false
  final_snapshot_identifier  = "${local.name}-db-final"
  auto_minor_version_upgrade = true
  apply_immediately          = true

  tags = { Name = "${local.name}-db" }
}
