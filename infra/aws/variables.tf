variable "project" {
  type    = string
  default = "thesis-bench"
}

variable "owner_email" {
  description = "adres na alarmy budżetu (tag Owner)"
  type        = string
}

variable "region" {
  type    = string
  default = "eu-central-1"
}

variable "availability_zone" {
  type    = string
  default = "eu-central-1a"
}

variable "my_ip_cidr" {
  description = "Twój publiczny adres IP z maską /32, np. 203.0.113.7/32 (start.sh ustawia go automatycznie)"
  type        = string
}

variable "ssh_public_key_path" {
  type    = string
  default = "~/.ssh/id_ed25519.pub"
}

variable "server_type" {
  description = "aplikacja (limit 2 CPU) + PostgreSQL (limit 3 CPU) w Dockerze, zapas na docker stats i system"
  type        = string
  default     = "c7i.2xlarge"
}

variable "generator_type" {
  description = "k6; zapas, żeby generator nie był wąskim gardłem przy ~10 tys. RPS z TLS"
  type        = string
  default     = "c7i.4xlarge"
}

variable "server_disk_gb" {
  type    = number
  default = 40
}

variable "generator_disk_gb" {
  type    = number
  default = 120
}

variable "repo_url" {
  type    = string
  default = "https://github.com/SzupanBurliga/masters-thesis.git"
}

variable "repo_ref" {
  description = "tag lub commit z zamrożonym kodem (np. v1.0-experiments)"
  type        = string
  default     = "main"
}

variable "budget_usd" {
  description = "miesięczny limit budżetu (alarmy przy 50/80/100%)"
  type        = number
  default     = 150
}
