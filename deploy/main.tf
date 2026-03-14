terraform {
  required_version = ">= 1.5"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.region
}

variable "region" {
  default = "us-east-1"
}

variable "environment" {
  default = "dev"
}

# VPC
resource "aws_vpc" "main" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true

  tags = {
    Name = "tradeengine-${var.environment}"
  }
}

resource "aws_subnet" "private" {
  count             = 2
  vpc_id            = aws_vpc.main.id
  cidr_block        = "10.0.${count.index + 1}.0/24"
  availability_zone = data.aws_availability_zones.available.names[count.index]

  tags = {
    Name = "tradeengine-private-${count.index + 1}"
  }
}

data "aws_availability_zones" "available" {
  state = "available"
}

# RDS (PostgreSQL)
resource "aws_db_instance" "postgres" {
  identifier           = "tradeengine-${var.environment}"
  engine               = "postgres"
  engine_version       = "15"
  instance_class       = "db.t3.micro"
  allocated_storage    = 20
  db_name              = "tradeengine"
  username             = "trade"
  password             = "CHANGE_ME_IN_SECRETS_MANAGER"
  skip_final_snapshot  = true
  vpc_security_group_ids = []

  tags = {
    Environment = var.environment
  }
}

# ElastiCache (Redis)
resource "aws_elasticache_cluster" "redis" {
  cluster_id           = "tradeengine-${var.environment}"
  engine               = "redis"
  node_type            = "cache.t3.micro"
  num_cache_nodes      = 1
  port                 = 6379

  tags = {
    Environment = var.environment
  }
}

# MSK (Kafka)
resource "aws_msk_cluster" "kafka" {
  cluster_name           = "tradeengine-${var.environment}"
  kafka_version          = "3.5.1"
  number_of_broker_nodes = 2

  broker_node_group_info {
    instance_type   = "kafka.t3.small"
    client_subnets  = aws_subnet.private[*].id
    storage_info {
      ebs_storage_info {
        volume_size = 20
      }
    }
  }

  tags = {
    Environment = var.environment
  }
}

# ECS Cluster
resource "aws_ecs_cluster" "main" {
  name = "tradeengine-${var.environment}"
}

# Outputs
output "rds_endpoint" {
  value = aws_db_instance.postgres.endpoint
}

output "redis_endpoint" {
  value = aws_elasticache_cluster.redis.cache_nodes[0].address
}

output "kafka_brokers" {
  value = aws_msk_cluster.kafka.bootstrap_brokers_tls
}
