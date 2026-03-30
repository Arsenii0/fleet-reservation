provider "aws" {
  region = var.aws_region

  default_tags {
    tags = {
      Project     = "openclaw-guardian"
      Component   = "openclaw"
      Environment = var.environment
      ManagedBy   = "Terraform"
    }
  }
}
