# ─────────────────────────────────────────────────────────────────────────────
# Data sources
# ─────────────────────────────────────────────────────────────────────────────

data "aws_partition" "current" {}
data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

# ─────────────────────────────────────────────────────────────────────────────
# Auto-generate VNC password (8 chars alphanumeric – TigerVNC max is 8)
# ─────────────────────────────────────────────────────────────────────────────

resource "random_password" "vnc" {
  length  = 8
  special = false
}

# Latest Ubuntu 24.04 LTS (Noble) HVM/SSD AMI
data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"] # Canonical

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd-gp3/ubuntu-noble-24.04-amd64-server-*"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }

  filter {
    name   = "architecture"
    values = ["x86_64"]
  }
}

# ToDo ArsenP: in production make it private subnet with additional NAT gateway for outbound access
resource "aws_vpc" "openclaw" {
  cidr_block           = var.vpc_cidr
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = merge(var.tags, { Name = "${var.name_prefix}-vpc" })
}

resource "aws_internet_gateway" "openclaw" {
  vpc_id = aws_vpc.openclaw.id

  tags = merge(var.tags, { Name = "${var.name_prefix}-igw" })
}

resource "aws_subnet" "openclaw_public" {
  vpc_id                  = aws_vpc.openclaw.id
  cidr_block              = var.public_subnet_cidr
  availability_zone       = var.availability_zone
  map_public_ip_on_launch = true

  tags = merge(var.tags, { Name = "${var.name_prefix}-public-subnet" })
}

resource "aws_route_table" "openclaw_public" {
  vpc_id = aws_vpc.openclaw.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.openclaw.id
  }

  tags = merge(var.tags, { Name = "${var.name_prefix}-public-rt" })
}

resource "aws_route_table_association" "openclaw_public" {
  subnet_id      = aws_subnet.openclaw_public.id
  route_table_id = aws_route_table.openclaw_public.id
}
