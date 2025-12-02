module "lambda" {
  source               = "./modules/lambda"
  environment          = var.environment
  contact              = var.contact
  product              = var.product
  orchestration        = var.orchestration
  distribution_bucket  = var.distribution_bucket
}
