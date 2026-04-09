# Copyright © 2026 Joseph Wright <joseph@cloudboss.co>
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in
# all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
# THE SOFTWARE.

locals {
  oidc_issuer = "https://${aws_s3_bucket.it.bucket_regional_domain_name}"
}

resource "aws_s3_bucket" "it" {
  bucket_prefix = "keights-${var.cluster_name}-oidc-"
  tags          = var.tags
}

resource "aws_s3_bucket_public_access_block" "it" {
  bucket = aws_s3_bucket.it.id

  block_public_acls       = true
  block_public_policy     = false
  ignore_public_acls      = true
  restrict_public_buckets = false
}

resource "aws_s3_bucket_policy" "it" {
  bucket = aws_s3_bucket.it.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect    = "Allow"
        Principal = "*"
        Action    = "s3:GetObject"
        Resource = [
          "${aws_s3_bucket.it.arn}/.well-known/openid-configuration",
          "${aws_s3_bucket.it.arn}/openid/v1/jwks",
        ]
      },
    ]
  })

  depends_on = [aws_s3_bucket_public_access_block.it]
}

resource "aws_s3_object" "oidc_discovery" {
  bucket       = aws_s3_bucket.it.bucket
  key          = ".well-known/openid-configuration"
  content_type = "application/json"
  content = jsonencode({
    issuer                                = local.oidc_issuer
    jwks_uri                              = "${local.oidc_issuer}/openid/v1/jwks"
    authorization_endpoint                = "urn:kubernetes:programmatic_authorization"
    response_types_supported              = ["id_token"]
    subject_types_supported               = ["public"]
    id_token_signing_alg_values_supported = ["RS256", "ES256"]
  })
}

resource "aws_s3_object" "oidc_jwks" {
  bucket       = aws_s3_bucket.it.bucket
  key          = "openid/v1/jwks"
  content_type = "application/json"
  content      = var.jwks
}

data "tls_certificate" "it" {
  url = local.oidc_issuer

  depends_on = [
    aws_s3_object.oidc_discovery,
    aws_s3_object.oidc_jwks,
  ]
}

resource "aws_iam_openid_connect_provider" "it" {
  url            = local.oidc_issuer
  client_id_list = [var.audience]
  thumbprint_list = [
    data.tls_certificate.it.certificates[0].sha1_fingerprint,
  ]
  tags = var.tags
}
