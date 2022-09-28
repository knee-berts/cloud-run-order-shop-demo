#!/usr/bin/env bash

set -Eeuo pipefail

script_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" &>/dev/null && pwd -P)

while getopts p:r:s:d: flag
do
    case "${flag}" in
        p) PROJECT_ID=${OPTARG};;
        r) PRIMARY_REGION=${OPTARG};;
        s) REGIONS=${OPTARG};;
        d) SPANNER_CONFIG=${OPTARG};;
    esac
done

echo "::Variable set::"
echo "PROJECT_ID: ${PROJECT_ID}"
echo "PRIMARY_REGION: ${PRIMARY_REGION}"
echo "REGIONS:${REGIONS}"
echo "SPANNER_CONFIG:${SPANNER_CONFIG}"

# REGION=${CLUSTER_LOCATION:0:-2}
PROJECT_NUMBER=$(gcloud projects describe ${PROJECT_ID} --format="value(projectNumber)")
echo "PROJECT_NUMBER:${PROJECT_NUMBER}"

## Enable GCP APIs
gcloud services enable --project=${PROJECT_ID} \
    artifactregistry.googleapis.com \
    spanner.googleapis.com \
    compute.googleapis.com \
    cloudbuild.googleapis.com \
    run.googleapis.com \
    cloudscheduler.googleapis.com

## Create Spanner Instance, DB, and Tables.
if [[ $(gcloud spanner instances describe orders-${PROJECT_ID} --project ${PROJECT_ID}) ]]; then
    echo "Spanner Instance orders-${PROJECT_ID} already exists"
else
    gcloud spanner instances create orders-${PROJECT_ID} --project ${PROJECT_ID} --config=${SPANNER_CONFIG} \
        --description="orders-${PROJECT_ID}" --processing-units=500
fi

if [[ $(gcloud spanner databases describe orders-db --instance=orders-${PROJECT_ID} --project ${PROJECT_ID}) ]]; then
    echo "Spanner DB orders-db already exists"
else
    gcloud spanner databases create orders-db --project ${PROJECT_ID} \
        --instance=orders-${PROJECT_ID} \
        --database-dialect=GOOGLE_STANDARD_SQL \
        --ddl-file=../../tables.ddl
fi

## Setup GCLB, DNS and Cert for Serverless NEGs
if [[ $(gcloud compute addresses describe orders-app --global --project ${PROJECT_ID}) ]]; then
  echo "Orders App IP already exists."
else
  echo "Creating Orders App IP."
  gcloud compute addresses create --global orders-app --project ${PROJECT_ID}
fi

export GCLB_IP=$(gcloud compute addresses describe orders-app  --global --project ${PROJECT_ID} --format="value(address)")

cat <<EOF > orders-app-openapi.yaml
swagger: "2.0"
info:
  description: "Cloud Endpoints DNS"
  title: "Cloud Endpoints DNS"
  version: "1.0.0"
paths: {}
host: "orders-app.endpoints.${PROJECT_ID}.cloud.goog"
x-google-endpoints:
- name: "orders-app.endpoints.${PROJECT_ID}.cloud.goog"
  target: "${GCLB_IP}"
EOF
gcloud endpoints services deploy orders-app-openapi.yaml --project ${PROJECT_ID}

if [[ $(gcloud beta compute ssl-certificates describe orders-app --project ${PROJECT_ID}) ]]; then
    echo "Cert for the orders-app already exists"
else
    gcloud beta compute ssl-certificates create orders-app \
        --domains="orders-app.endpoints.${PROJECT_ID}.cloud.goog" --project ${PROJECT_ID}
fi

## Setup Service Accounts
SERVICE_ACCOUNTS=(
    "orders-web-sa"
    "orders-worker-sa"
    "orders-job-sa"
)

for SA in "${SERVICE_ACCOUNTS[@]}"; do
    if [[ ${SA} == "orders-worker-sa" ]]; then
        echo ${SA}
        if [[ $(gcloud iam service-accounts describe "${SA}@${PROJECT_ID}.iam.gserviceaccount.com" --project ${PROJECT_ID}) ]]; then
            echo "Service account ${SA} for the orders-app already exists"
        else
            gcloud iam service-accounts create ${SA} --project ${PROJECT_ID}
        fi
        gcloud projects add-iam-policy-binding ${PROJECT_ID} \
            --member "serviceAccount:${SA}@${PROJECT_ID}.iam.gserviceaccount.com" \
            --role "roles/spanner.databaseUser"

        gcloud projects add-iam-policy-binding ${PROJECT_ID} \
            --member "serviceAccount:${SA}@${PROJECT_ID}.iam.gserviceaccount.com" \
            --role "roles/pubsub.editor"
    else
        if [[ $(gcloud iam service-accounts describe "${SA}@${PROJECT_ID}.iam.gserviceaccount.com" --project ${PROJECT_ID}) ]]; then
            echo "Service account ${SA} for the orders-app already exists"
        else
            gcloud iam service-accounts create ${SA} --project ${PROJECT_ID}
        fi
        gcloud projects add-iam-policy-binding ${PROJECT_ID} \
            --member "serviceAccount:${SA}@${PROJECT_ID}.iam.gserviceaccount.com" \
            --role "roles/spanner.databaseUser"
    fi
done

## Setup Artifact Repository for Container images and build and push images
if [[ $(gcloud artifacts repositories describe orders-repo --location ${PRIMARY_REGION} --project ${PROJECT_ID}) ]]; then
    echo "Artifact Registry for the orders-app already exists"
else
    gcloud artifacts repositories create orders-repo --repository-format=docker \
        --location=${PRIMARY_REGION} --description="Docker repository" --project ${PROJECT_ID}
fi

gcloud auth configure-docker ${PRIMARY_REGION}-docker.pkg.dev --project ${PROJECT_ID}

gcloud builds submit --region=${PRIMARY_REGION} --project ${PROJECT_ID} --tag "${PRIMARY_REGION}-docker.pkg.dev/${PROJECT_ID}/orders-repo/orders-web" ../../run-web

gcloud builds submit --region=${PRIMARY_REGION} --project ${PROJECT_ID} --tag "${PRIMARY_REGION}-docker.pkg.dev/${PROJECT_ID}/orders-repo/orders-job" ../../run-job

gcloud builds submit --region=${PRIMARY_REGION} --project ${PROJECT_ID} --tag "${PRIMARY_REGION}-docker.pkg.dev/${PROJECT_ID}/orders-repo/orders-worker" ../../run-worker


export SPANNER_URI="projects/${PROJECT_ID}/instances/orders-${PROJECT_ID}/databases/orders-db"

## Deploy Worker in Primary Region
if [[ $(gcloud run services describe orders-worker-${PRIMARY_REGION} --region ${PRIMARY_REGION} --project ${PROJECT_ID}) ]]; then
    echo "Cloud Run servive orders-worker-${PRIMARY_REGION} already exists"
else
    gcloud run deploy orders-worker-${PRIMARY_REGION} --project ${PROJECT_ID} \
        --min-instances=1 \
        --no-cpu-throttling \
        --ingress=internal \
        --service-account="orders-worker-sa@${PROJECT_ID}.iam.gserviceaccount.com" \
        --image="${PRIMARY_REGION}-docker.pkg.dev/${PROJECT_ID}/orders-repo/orders-worker" \
        --set-env-vars="SPANNER_URI=${SPANNER_URI}, APP_PORT=8080" \
        --region=${PRIMARY_REGION} -q
fi


## Deploy Cron Job in Primary Region
if [[ $(gcloud beta run jobs describe orders-job-${PRIMARY_REGION} --region ${PRIMARY_REGION} --project ${PROJECT_ID}) ]]; then
    echo "Cloud Run servive create orders-job-${PRIMARY_REGION} already exists"
else
    gcloud beta run jobs create orders-job-${PRIMARY_REGION} --project ${PROJECT_ID}\
        --service-account="orders-job-sa@${PROJECT_ID}.iam.gserviceaccount.com" \
        --image="${PRIMARY_REGION}-docker.pkg.dev/${PROJECT_ID}/orders-repo/orders-job" \
        --set-env-vars="SPANNER_URI=${SPANNER_URI}" \
        --region=${PRIMARY_REGION} -q
fi

if [[ $(gcloud scheduler jobs describe orders-job-${PRIMARY_REGION} --location ${PRIMARY_REGION} --project ${PROJECT_ID}) ]]; then
    echo "Scheduler for orders-job-${PRIMARY_REGION} already exists"
else
    gcloud scheduler jobs create http orders-job-${PRIMARY_REGION} --project ${PROJECT_ID} \
        --location ${PRIMARY_REGION} \
        --schedule="*/5 * * * *" \
        --uri="https://${PRIMARY_REGION}-run.googleapis.com/apis/run.googleapis.com/v1/namespaces/${PROJECT_ID}/jobs/orders-job-${PRIMARY_REGION}:run" \
        --http-method POST \
        --oauth-service-account-email ${PROJECT_NUMBER}-compute@developer.gserviceaccount.com
fi

echo "Build completed. Give the GCLB a minute to configure endpoints."
echo "After a bit the global orders web link will load https://orders-app.endpoints.${PROJECT_ID}.cloud.goog"
echo "Generate a random order: curl -X PUT https://orders-app.endpoints.${PROJECT_ID}.cloud.goog/addRandomOrder"
echo "Get the count of submitted orders: curl https://orders-app.endpoints.${PROJECT_ID}.cloud.goog/orderStatusCount/SUBMITTED"