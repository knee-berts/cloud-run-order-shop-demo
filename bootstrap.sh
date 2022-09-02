export PROJECT_ID=cloud-code-demo-360914
PROJECT_NUMBER=$(gcloud projects describe ${PROJECT_ID} --format="value(projectNumber)")
export SPANNER_URI="projects/cloud-code-demo-360914/instances/nick-spanner/databases/nickdb"
gcloud compute addresses create --global orders-app --project ${PROJECT_ID}

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

gcloud beta compute ssl-certificates create orders-app \
  --domains="orders-app.endpoints.${PROJECT_ID}.cloud.goog"

gcloud compute backend-services create --global orders-app
gcloud compute url-maps create orders-app --default-service=orders-app
gcloud compute target-https-proxies create orders-app \
  --ssl-certificates=orders-app \
  --url-map=orders-app
gcloud compute forwarding-rules create --global orders-app \
  --target-https-proxy=orders-app \
  --address=orders-app \
  --ports=443

gcloud iam service-accounts create orders-sa \
  --project=cloud-code-demo-360914

gcloud projects add-iam-policy-binding cloud-code-demo-360914 \
    --member "serviceAccount:orders-sa@cloud-code-demo-360914.iam.gserviceaccount.com" \
    --role "roles/spanner.databaseUser"

gcloud projects add-iam-policy-binding cloud-code-demo-360914 \
    --member "serviceAccount:orders-sa@cloud-code-demo-360914.iam.gserviceaccount.com" \
    --role "roles/pubsub.editor"


gcloud iam service-accounts add-iam-policy-binding orders-sa@cloud-code-demo-360914.iam.gserviceaccount.com \
    --role roles/iam.workloadIdentityUser \
    --member "serviceAccount:cloud-code-demo-360914.svc.id.goog[default/default]"

gcloud run deploy orders-frontend-us-west1 \
    --allow-unauthenticated \
    --image=us-east1-docker.pkg.dev/cloud-code-demo-360914/repo/orders-frontend \
    --service-account="orders-sa@cloud-code-demo-360914.iam.gserviceaccount.com" \
    --region=us-west1
gcloud beta compute network-endpoint-groups create orders-us-west1 \
    --region=us-west1 \
    --network-endpoint-type=SERVERLESS \
    --cloud-run-service=orders-frontend-us-west1 
gcloud beta compute backend-services add-backend --global orders-app \
    --network-endpoint-group-region=us-west1 \
    --network-endpoint-group=orders-us-west1

cd run-worker
gcloud builds submit --pack image=us-east1-docker.pkg.dev/${PROJECT_ID}/repo/orders-worker
gcloud run deploy orders-worker-us-east1 \
    --min-instances=1 \
    --no-cpu-throttling \
    --ingress=internal \
    --service-account="orders-sa@cloud-code-demo-360914.iam.gserviceaccount.com" \
    --image=us-east1-docker.pkg.dev/cloud-code-demo-360914/repo/orders-worker \
    --set-env-vars="SPANNER_URI=${SPANNER_URI}, APP_PORT=8080" \
    --region=us-east1
# gcloud beta compute network-endpoint-groups create orders-us-east1 \
#     --region=us-east1 \
#     --network-endpoint-type=SERVERLESS \
#     --cloud-run-service=orders-frontend-us-east1
# gcloud beta compute backend-services add-backend --global orders-app \
#     --network-endpoint-group-region=us-east1 \
#     --network-endpoint-group=orders-us-east1

cd run-job
gcloud beta run jobs create orders-job-us-east1 \
    --service-account="orders-sa@cloud-code-demo-360914.iam.gserviceaccount.com" \
    --image=us-east1-docker.pkg.dev/cloud-code-demo-360914/repo/orders-job \
    --set-env-vars="SPANNER_URI=${SPANNER_URI}" \
    --region=us-east1

gcloud scheduler jobs create http orders-job-us-east1 \
  --location us-east1 \
    --schedule="*/30 * * * *" \
  --uri="https://us-east1-run.googleapis.com/apis/run.googleapis.com/v1/namespaces/${PROJECT_ID}/jobs/orders-job-us-east1:run" \
  --http-method POST \
  --oauth-service-account-email ${PROJECT_NUMBER}-compute@developer.gserviceaccount.com
    # --tasks 100 \
    # --parallelism 100 
