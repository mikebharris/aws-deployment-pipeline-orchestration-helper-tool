output "endpoint_url" {
  value = aws_lambda_function_url.hello_world_lambda_function_url.function_url
}