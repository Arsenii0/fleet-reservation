# ─────────────────────────────────────────────────────────────────────────────
# EC2 instance profile – minimal permissions:
#   • SSM Session Manager (replaces inbound SSH)
# The instance has NO S3 / EBS / EKS access – principle of least privilege.
# ─────────────────────────────────────────────────────────────────────────────

resource "aws_iam_role" "openclaw_ec2" {
  name = "${var.name_prefix}-ec2-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "ec2.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })

  tags = merge(var.tags, { Name = "${var.name_prefix}-ec2-role" })
}

# Allow SSM Session Manager to open a shell without any inbound port
resource "aws_iam_role_policy_attachment" "ssm_core" {
  role       = aws_iam_role.openclaw_ec2.name
  policy_arn = "arn:${data.aws_partition.current.partition}:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

resource "aws_iam_instance_profile" "openclaw_ec2" {
  name = "${var.name_prefix}-ec2-profile"
  role = aws_iam_role.openclaw_ec2.name

  tags = merge(var.tags, { Name = "${var.name_prefix}-ec2-profile" })
}
