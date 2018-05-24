# stackhand

A collection of CloudFormation custom resource Lambdas written in Go.

[eawsy/aws-lambda-go-shim](https://github.com/eawsy/aws-lambda-go-shim) is used to provide a shim for running Go on the Python 2.7 runtime.

# Lambdas

### subnet_to_az

`subnet_to_az` retrieves the availability zone from a given subnet.

Handler:

`handler.Handle`

Properties:

`Region`: AWS region to search

`SubnetId`: ID of subnet to search for availability zone

Returns:

```
{
  "AvailabilityZone": "..."
}
```

Hosted at s3://cloudboss-public/lambda/co/cloudboss/stackhand/subnet_to_az/0.1.0/python2.7/subnet_to_az-0.1.0.zip.

CloudFormation example:

```
Parameters:
  SubnetId:
    Description: Subnet ID in VPC where cluster will be placed
    Type: AWS::EC2::Subnet::Id

Resources:
  # Deploy the Lambda
  SubnetToAzFunction:
    Type: AWS::Lambda::Function
    Properties:
      Code:
        S3Bucket: cloudboss-public
        S3Key: lambda/co/cloudboss/stackhand/subnet_to_az/0.1.0/python2.7/subnet_to_az-0.1.0.zip
      Handler: handler.Handle
      Runtime: python2.7
      Timeout: 30
      Role: !Ref LambdaExecutionRole

  # Run the Lambda as a custom resource
  SubnetToAz:
    Type: Custom::SubnetToAz
    DependsOn: SubnetToAzFunction
    Properties:
      ServiceToken: !GetAtt [SubnetToAzFunction, Arn]
      Region: !Ref AWS::Region
      SubnetId: !Ref SubnetId

  # Use its output with !GetAtt
  Volume:
    Type: AWS::EC2::Volume
    Properties:
      AvailabilityZone: !GetAtt [SubnetToAz, AvailabilityZone]
      Size: 10
      VolumeType: gp2
```
