# Google Cloud AI

## Vertex AI
`gcloud ai models upload` upload model
`gcloud ai endpoints deploy-model` deploy model to endpoint
`gcloud ai endpoints predict` send prediction request
`gcloud ai custom-jobs create` run custom training job

## Agent Builder (Gen App Builder)
*Currently primarily managed via Console or API*

## Common Regions
`us-central1` Iowa
`us-east1` South Carolina
`us-west1` Oregon
`europe-west1` Belgium
`asia-east1` Taiwan

## Setup
1. Install gcloud CLI: https://cloud.google.com/sdk/docs/install
2. `gcloud auth login`
3. `gcloud config set project PROJECT_ID`
4. Enable Vertex AI API: `gcloud services enable aiplatform.googleapis.com`

## Sources
- https://cloud.google.com/sdk/gcloud/reference/ai
- https://cloud.google.com/vertex-ai/docs/start/cli
