# ─────────────────────────────────────────────────────────────────────────────
# KMS key for EBS encryption
# ─────────────────────────────────────────────────────────────────────────────

resource "aws_kms_key" "openclaw_ebs" {
  description             = "OpenClaw EC2 EBS encryption key"
  deletion_window_in_days = 7
  enable_key_rotation     = true

  tags = merge(var.tags, { Name = "${var.name_prefix}-ebs-key" })
}

resource "aws_kms_alias" "openclaw_ebs" {
  name          = "alias/${var.name_prefix}-ebs"
  target_key_id = aws_kms_key.openclaw_ebs.key_id
}

# ─────────────────────────────────────────────────────────────────────────────
# EC2 Instance
# ─────────────────────────────────────────────────────────────────────────────

resource "aws_instance" "openclaw" {
  ami           = data.aws_ami.ubuntu.id
  instance_type = var.instance_type
  subnet_id     = aws_subnet.openclaw_public.id

  iam_instance_profile = aws_iam_instance_profile.openclaw_ec2.name

  vpc_security_group_ids = [aws_security_group.openclaw.id]

  # Require IMDSv2 on the metadata service (prevents SSRF credential theft)
  metadata_options {
    http_tokens                 = "required"  # IMDSv2 only
    http_put_response_hop_limit = 1           # blocks Docker container escapes
    instance_metadata_tags      = "enabled"
  }

  # Encrypted root volume
  root_block_device {
    volume_type           = "gp3"
    volume_size           = var.root_volume_size_gb
    encrypted             = true
    kms_key_id            = aws_kms_key.openclaw_ebs.arn
    delete_on_termination = true

    tags = merge(var.tags, { Name = "${var.name_prefix}-root-vol" })
  }

  # Disable accidental public IP assignment (Elastic IP ensures stable IP across stop/start).
  associate_public_ip_address = false

  # Bootstrap script
  user_data = templatefile("${path.module}/user_data.sh", {
    name_prefix           = var.name_prefix
    openclaw_version      = var.openclaw_version
    vnc_password          = random_password.vnc.result
    vnc_allowed_cidr      = var.vnc_allowed_cidr
  })

  tags = merge(var.tags, { Name = "${var.name_prefix}" })
}

# Elastic IP – same IP address across stop/start cycles
resource "aws_eip" "openclaw" {
  domain   = "vpc"
  instance = aws_instance.openclaw.id

  tags = merge(var.tags, { Name = "${var.name_prefix}-eip" })
}
