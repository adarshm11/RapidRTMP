# Google Cloud Storage (GCS) Setup Guide

This guide explains how to configure RapidRTMP to use Google Cloud Storage for storing HLS segments and playlists.

## Benefits of Using GCS

✅ **Scalability** - Unlimited storage capacity  
✅ **Durability** - 99.999999999% (11 9's) durability  
✅ **CDN Integration** - Easy integration with Cloud CDN  
✅ **Global Distribution** - Multi-regional storage options  
✅ **Cost-Effective** - Pay only for what you use  
✅ **Performance** - Low-latency access worldwide  

---

## Prerequisites

1. **Google Cloud Platform Account**
   - Sign up at https://cloud.google.com/
   - Enable billing

2. **GCP Project**
   - Create or select a project
   - Note your Project ID

3. **gcloud CLI** (optional but recommended)
   ```bash
   # Install gcloud
   # https://cloud.google.com/sdk/docs/install
   
   # Initialize
   gcloud init
   ```

---

## Step 1: Create a GCS Bucket

### Using GCP Console

1. Go to [Cloud Storage Console](https://console.cloud.google.com/storage)
2. Click "CREATE BUCKET"
3. Configure:
   - **Name**: `rapidrtmp-streams` (must be globally unique)
   - **Location type**: Choose based on your needs:
     - **Region**: Single region (lowest latency for specific region)
     - **Multi-region**: US, EU, or ASIA (higher availability)
   - **Storage class**: Standard (best for hot data)
   - **Access control**: Uniform (recommended)
4. Click "CREATE"

### Using gcloud CLI

```bash
# Create bucket in us-central1
gsutil mb -c STANDARD -l us-central1 gs://rapidrtmp-streams

# Or create multi-regional bucket
gsutil mb -c STANDARD -l US gs://rapidrtmp-streams
```

---

## Step 2: Set Up Service Account

### Create Service Account

1. Go to [IAM & Admin > Service Accounts](https://console.cloud.google.com/iam-admin/serviceaccounts)
2. Click "CREATE SERVICE ACCOUNT"
3. Enter details:
   - **Name**: `rapidrtmp-storage`
   - **Description**: "RapidRTMP storage access"
4. Click "CREATE AND CONTINUE"

### Grant Permissions

Add these roles:
- **Storage Object Admin** (`roles/storage.objectAdmin`)
  - Full control over objects in the bucket

Click "CONTINUE" → "DONE"

### Create Key

1. Click on the service account
2. Go to "KEYS" tab
3. Click "ADD KEY" → "Create new key"
4. Choose "JSON"
5. Download the key file (e.g., `rapidrtmp-storage-key.json`)

⚠️ **Keep this file secure!** It provides full access to your storage.

### Using gcloud CLI

```bash
# Create service account
gcloud iam service-accounts create rapidrtmp-storage \
    --description="RapidRTMP storage access" \
    --display-name="RapidRTMP Storage"

# Grant permissions
gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
    --member="serviceAccount:rapidrtmp-storage@YOUR_PROJECT_ID.iam.gserviceaccount.com" \
    --role="roles/storage.objectAdmin"

# Create and download key
gcloud iam service-accounts keys create rapidrtmp-storage-key.json \
    --iam-account=rapidrtmp-storage@YOUR_PROJECT_ID.iam.gserviceaccount.com
```

---

## Step 3: Configure RapidRTMP

### Environment Variables

Set these environment variables:

```bash
# Storage Configuration
export STORAGE_TYPE="gcs"
export GCS_PROJECT_ID="your-project-id"
export GCS_BUCKET_NAME="rapidrtmp-streams"
export GCS_BASE_DIR="streams"  # Optional: base directory within bucket

# Google Cloud Credentials
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/rapidrtmp-storage-key.json"

# Other configuration
export HTTP_ADDR=":8080"
export RTMP_ADDR=":1935"
```

### Configuration File (Optional)

Create `.env` file:

```bash
STORAGE_TYPE=gcs
GCS_PROJECT_ID=your-project-id
GCS_BUCKET_NAME=rapidrtmp-streams
GCS_BASE_DIR=streams
GOOGLE_APPLICATION_CREDENTIALS=./rapidrtmp-storage-key.json
HTTP_ADDR=:8080
RTMP_ADDR=:1935
```

---

## Step 4: Run RapidRTMP with GCS

### Local Development

```bash
# Set credentials
export GOOGLE_APPLICATION_CREDENTIALS="/path/to/rapidrtmp-storage-key.json"

# Set GCS configuration
export STORAGE_TYPE="gcs"
export GCS_PROJECT_ID="your-project-id"
export GCS_BUCKET_NAME="rapidrtmp-streams"

# Run
./rapidrtmp
```

### Docker

```bash
docker run -d \
  -p 8080:8080 \
  -p 1935:1935 \
  -e STORAGE_TYPE=gcs \
  -e GCS_PROJECT_ID=your-project-id \
  -e GCS_BUCKET_NAME=rapidrtmp-streams \
  -e GOOGLE_APPLICATION_CREDENTIALS=/secrets/gcs-key.json \
  -v /path/to/rapidrtmp-storage-key.json:/secrets/gcs-key.json:ro \
  rapidrtmp:latest
```

### Kubernetes

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: gcs-credentials
type: Opaque
data:
  key.json: <base64-encoded-service-account-key>
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: rapidrtmp
spec:
  template:
    spec:
      containers:
      - name: rapidrtmp
        image: rapidrtmp:latest
        env:
        - name: STORAGE_TYPE
          value: "gcs"
        - name: GCS_PROJECT_ID
          value: "your-project-id"
        - name: GCS_BUCKET_NAME
          value: "rapidrtmp-streams"
        - name: GCS_BASE_DIR
          value: "streams"
        - name: GOOGLE_APPLICATION_CREDENTIALS
          value: "/secrets/gcs/key.json"
        volumeMounts:
        - name: gcs-credentials
          mountPath: /secrets/gcs
          readOnly: true
      volumes:
      - name: gcs-credentials
        secret:
          secretName: gcs-credentials
```

---

## Step 5: Enable Cloud CDN (Optional)

For global distribution and caching:

### 1. Create Load Balancer

```bash
# Reserve external IP
gcloud compute addresses create rapidrtmp-ip --global

# Create backend bucket
gcloud compute backend-buckets create rapidrtmp-backend \
    --gcs-bucket-name=rapidrtmp-streams \
    --enable-cdn
```

### 2. Configure URL Map

```bash
# Create URL map
gcloud compute url-maps create rapidrtmp-lb \
    --default-backend-bucket=rapidrtmp-backend

# Create HTTP proxy
gcloud compute target-http-proxies create rapidrtmp-proxy \
    --url-map=rapidrtmp-lb

# Create forwarding rule
gcloud compute forwarding-rules create rapidrtmp-http-rule \
    --global \
    --target-http-proxy=rapidrtmp-proxy \
    --address=rapidrtmp-ip \
    --ports=80
```

### 3. Get CDN URL

```bash
gcloud compute addresses describe rapidrtmp-ip --global
```

---

## Monitoring & Costs

### View Storage Metrics

```bash
# Check bucket size
gsutil du -sh gs://rapidrtmp-streams

# List objects
gsutil ls -r gs://rapidrtmp-streams

# View bucket details
gcloud storage buckets describe gs://rapidrtmp-streams
```

### Cost Optimization

1. **Lifecycle Policies** - Auto-delete old segments:

```bash
# Create lifecycle config
cat > lifecycle.json <<EOF
{
  "lifecycle": {
    "rule": [
      {
        "action": {"type": "Delete"},
        "condition": {
          "age": 7,
          "matchesPrefix": ["streams/"]
        }
      }
    ]
  }
}
EOF

# Apply lifecycle policy
gsutil lifecycle set lifecycle.json gs://rapidrtmp-streams
```

2. **Storage Classes**:
   - **Standard**: Best for hot data (< 30 days)
   - **Nearline**: Accessed < once/month (saves 50%)
   - **Coldline**: Accessed < once/quarter (saves 70%)

---

## Troubleshooting

### Error: "Failed to create GCS client"

**Solution**: Check credentials are set correctly
```bash
echo $GOOGLE_APPLICATION_CREDENTIALS
cat $GOOGLE_APPLICATION_CREDENTIALS  # Verify file exists
```

### Error: "Failed to access bucket"

**Solution**: Verify bucket name and permissions
```bash
# Test access
gsutil ls gs://your-bucket-name

# Check IAM permissions
gcloud projects get-iam-policy YOUR_PROJECT_ID
```

### Error: "Permission denied"

**Solution**: Ensure service account has correct roles
```bash
gcloud projects add-iam-policy-binding YOUR_PROJECT_ID \
    --member="serviceAccount:rapidrtmp-storage@YOUR_PROJECT_ID.iam.gserviceaccount.com" \
    --role="roles/storage.objectAdmin"
```

---

## Security Best Practices

1. **Use Service Accounts** - Don't use personal credentials
2. **Principle of Least Privilege** - Grant minimum required permissions
3. **Rotate Keys** - Regularly rotate service account keys
4. **Bucket Access Control** - Use uniform access control
5. **VPC Service Controls** - Restrict access to authorized networks
6. **Audit Logs** - Enable Cloud Audit Logs for monitoring

---

## Performance Tips

1. **Choose Nearest Region** - Select region closest to your users
2. **Use Cloud CDN** - Cache content globally
3. **Parallel Uploads** - GCS handles concurrent writes well
4. **Compression** - Enable gzip compression for playlists
5. **Connection Pooling** - Reuse HTTP connections

---

## Cost Estimates

**Example: 100 concurrent streams, 10 viewers each**

- **Storage**: ~50 GB/month @ $0.020/GB = $1
- **Network Egress**: ~10 TB/month @ $0.12/GB = $1,200
- **Operations**: 100M Class A @ $0.05/10k = $500

**Total**: ~$1,701/month

💡 **Tip**: Use Cloud CDN to reduce network egress costs by 50-80%!

---

## Support

- [GCS Documentation](https://cloud.google.com/storage/docs)
- [Pricing Calculator](https://cloud.google.com/products/calculator)
- [Status Dashboard](https://status.cloud.google.com/)

