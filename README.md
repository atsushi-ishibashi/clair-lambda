# clair-lambda
clair-lambda is a lambda function for vulnerability static analysis for containers with clair.

## Overview
The lambda function is invoked by CloudWatchEvent and send vulnerability analysis requests to clair server.

## Assumption
- clair API server(ver.1)
- `ecr:GetAuthorizationToken` is allowed to the lambda
- the below CloudWatchEvent
```
{
  "source": [
    "aws.ecr"
  ],
  "detail-type": [
    "AWS API Call via CloudTrail"
  ],
  "detail": {
    "eventSource": [
      "ecr.amazonaws.com"
    ],
    "eventName": [
      "PutImage"
    ]
  }
}
```

### Related
[clair](https://github.com/coreos/clair)  
[klar](https://github.com/optiopay/klar)
