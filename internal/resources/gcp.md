# Google Cloud

`gcloud auth login` log in to GCP ^run
`gcloud config set project {{project}}` set active project ^run:project
`gcloud config list` show configuration ^run
`gcloud projects list` list projects ^run
`gcloud services list --enabled` list enabled services ^run
`gcloud services enable aiplatform.googleapis.com` enable Vertex AI API ^run
`gcloud ai models list --region={{region}}` list Vertex AI models ^run:region
`gcloud ai endpoints list --region={{region}}` list Vertex AI endpoints ^run:region
`gcloud ai custom-jobs list --region={{region}}` list custom training jobs ^run:region
`gcloud ai index-endpoints list --region={{region}}` list vector search endpoints ^run:region
`gcloud run services list` list Cloud Run services (agents) ^run
`gcloud compute instances list` list VM instances ^run
`gcloud container clusters list` list GKE clusters ^run
