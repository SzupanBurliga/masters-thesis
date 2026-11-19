# Środowisko pomiarowe AWS: generator k6 + serwer (aplikacja i PostgreSQL w Dockerze, te same limity co lokalnie).
# NIE URUCHAMIAĆ bez akceptacji kosztów (README.md w tym katalogu). Sprzątanie: ./teardown.sh

terraform {
  required_version = ">= 1.6"
  required_providers {
    aws = { source = "hashicorp/aws", version = "~> 5.80" }
  }
}

provider "aws" {
  region = var.region
  default_tags {
    tags = { Project = var.project, Owner = var.owner_email }
  }
}

data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"] # Canonical
  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-amd64-server-*"]
  }
}

data "aws_vpc" "default" { default = true }

data "aws_subnet" "az" {
  vpc_id            = data.aws_vpc.default.id
  availability_zone = var.availability_zone
  default_for_az    = true
}

# Obie instancje w jednej strefie i w placement group "cluster": najniższe i najstabilniejsze opóźnienie sieci.
resource "aws_placement_group" "bench" {
  name     = "${var.project}-cluster"
  strategy = "cluster"
}

resource "aws_key_pair" "me" {
  key_name   = "${var.project}-key"
  public_key = file(pathexpand(var.ssh_public_key_path))
}

resource "aws_security_group" "bench" {
  name        = "${var.project}-sg"
  description = "SSH tylko z mojego IP; ruch testowy tylko miedzy instancjami tej grupy"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    description = "SSH z mojego IP"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [var.my_ip_cidr]
  }
  ingress {
    description = "ruch miedzy generatorem a serwerem (8443-8445, 9101-9103, 5432)"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    self        = true
  }
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

locals {
  common = {
    ami                    = data.aws_ami.ubuntu.id
    subnet_id              = data.aws_subnet.az.id
    key_name               = aws_key_pair.me.key_name
    vpc_security_group_ids = [aws_security_group.bench.id]
    placement_group        = aws_placement_group.bench.name
  }
}

resource "aws_instance" "server" {
  ami                    = local.common.ami
  instance_type          = var.server_type
  subnet_id              = local.common.subnet_id
  key_name               = local.common.key_name
  vpc_security_group_ids = local.common.vpc_security_group_ids
  placement_group        = local.common.placement_group
  user_data              = templatefile("${path.module}/user_data.sh", { role = "server", repo_url = var.repo_url, repo_ref = var.repo_ref })
  root_block_device {
    volume_type = "gp3"
    volume_size = var.server_disk_gb
  }
  tags = { Name = "${var.project}-server", Role = "server" }
}

resource "aws_instance" "generator" {
  ami                    = local.common.ami
  instance_type          = var.generator_type
  subnet_id              = local.common.subnet_id
  key_name               = local.common.key_name
  vpc_security_group_ids = local.common.vpc_security_group_ids
  placement_group        = local.common.placement_group
  user_data              = templatefile("${path.module}/user_data.sh", { role = "generator", repo_url = var.repo_url, repo_ref = var.repo_ref })
  root_block_device {
    volume_type = "gp3"
    volume_size = var.generator_disk_gb # *_points.csv.gz: rząd 30-70 GB dla całej macierzy
  }
  tags = { Name = "${var.project}-generator", Role = "generator" }
}

# Budżet miesięczny z alarmami e-mail (50%, 80%, 100% rzeczywistych i 100% prognozowanych kosztów).
resource "aws_budgets_budget" "monthly" {
  name         = "${var.project}-monthly"
  budget_type  = "COST"
  limit_amount = var.budget_usd
  limit_unit   = "USD"
  time_unit    = "MONTHLY"

  dynamic "notification" {
    for_each = [50, 80, 100]
    content {
      comparison_operator        = "GREATER_THAN"
      threshold                  = notification.value
      threshold_type             = "PERCENTAGE"
      notification_type          = "ACTUAL"
      subscriber_email_addresses = [var.owner_email]
    }
  }
  notification {
    comparison_operator        = "GREATER_THAN"
    threshold                  = 100
    threshold_type             = "PERCENTAGE"
    notification_type          = "FORECASTED"
    subscriber_email_addresses = [var.owner_email]
  }
}
