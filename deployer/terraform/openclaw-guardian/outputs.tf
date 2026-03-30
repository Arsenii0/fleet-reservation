output "instance_id" {
  description = "EC2 instance ID – use this to open an SSM session"
  value       = aws_instance.openclaw.id
}

output "instance_public_ip" {
  description = "Stable Elastic IP address of the OpenClaw instance"
  value       = aws_eip.openclaw.public_ip
}

output "instance_type" {
  description = "Instance type in use"
  value       = aws_instance.openclaw.instance_type
}

output "ami_id" {
  description = "AMI used (Ubuntu 24.04 LTS)"
  value       = data.aws_ami.ubuntu.id
}

output "ssm_start_session_cmd" {
  description = "Command to open an interactive shell via SSM Session Manager"
  value       = "aws ssm start-session --target ${aws_instance.openclaw.id} --region ${var.aws_region}"
}

output "ssm_port_forward_cmd" {
  description = "Command to port-forward the OpenClaw gateway WebUI to localhost:18789"
  value       = <<-EOT
    aws ssm start-session \
      --target ${aws_instance.openclaw.id} \
      --region ${var.aws_region} \
      --document-name AWS-StartPortForwardingSession \
      --parameters 'portNumber=18789,localPortNumber=18789'
    # Then open: http://localhost:18789
  EOT
}

output "security_group_id" {
  description = "Security group attached to the instance"
  value       = aws_security_group.openclaw.id
}

output "iam_role_arn" {
  description = "IAM role ARN attached to the instance"
  value       = aws_iam_role.openclaw_ec2.arn
}

# ── VNC credentials ───────────────────────────────────────────────────────────

output "vnc_username" {
  description = "OS user to connect as over VNC"
  value       = "ubuntu"
}

output "vnc_password" {
  description = "Auto-generated VNC password"
  value       = random_password.vnc.result
  sensitive   = true
}

output "vnc_connect" {
  description = "VNC connection string"
  value       = "${aws_eip.openclaw.public_ip}:5901"
}
