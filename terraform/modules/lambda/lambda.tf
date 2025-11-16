resource "aws_iam_role" "attendees_api_iam_role" {
  name                  = "${var.product}-${var.environment}-hello-world-iam-role"
  force_detach_policies = true
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Sid    = ""
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      }
    ]
  })
}

resource "aws_cloudwatch_log_group" "cloudwatch_log_group" {
  name              = "/aws/lambda/${aws_lambda_function.hello_world_lambda_function.function_name}"
  retention_in_days = 1
}

resource "aws_iam_role_policy_attachment" "hello_world_policy_attachment_execution" {
  role       = aws_iam_role.attendees_api_iam_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

data "archive_file" "hello_world_lambda_function_distribution" {
  source_file = "../lambdas/helloworld/bootstrap"
  output_path = "../lambdas/helloworld/helloworld.zip"
  type        = "zip"
}

resource "aws_s3_object" "hello_world_lambda_function_distribution_bucket_object" {
  bucket = var.distribution_bucket
  key    = "lambdas/${var.environment}/${var.product}/helloworld.zip"
  source = data.archive_file.hello_world_lambda_function_distribution.output_path
  etag   = filemd5(data.archive_file.hello_world_lambda_function_distribution.output_path)
}

resource "aws_lambda_function" "hello_world_lambda_function" {
  function_name    = "${var.environment}-${var.product}-hello-world"
  role             = aws_iam_role.attendees_api_iam_role.arn
  handler          = "bootstrap"
  runtime          = "provided.al2023"
  architectures    = ["arm64"]
  s3_bucket        = aws_s3_object.hello_world_lambda_function_distribution_bucket_object.bucket
  s3_key           = aws_s3_object.hello_world_lambda_function_distribution_bucket_object.key
  source_code_hash = data.archive_file.hello_world_lambda_function_distribution.output_md5
  timeout          = 15
  memory_size      = 128

  tags = {
    Environment   = var.environment
    Project       = var.product
    Contact       = var.contact
    Orchestration = var.orchestration
  }
}

resource "aws_lambda_function_url" "hello_world_lambda_function_url" {
  authorization_type = "NONE"
  function_name      = aws_lambda_function.hello_world_lambda_function.function_name
}
