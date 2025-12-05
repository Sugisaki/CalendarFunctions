#!/bin/bash

set -e  # 途中でエラーが出たら中断

SOURCE_DIR="./calendar"  # 関数コードの場所（相対パスOK）
FUNCTION_NAME="holidayapi"
ENTRY_POINT="HandleHolidayRequest"

echo "🔧 Running go mod tidy..."
(cd "$SOURCE_DIR" && go mod tidy)

echo "🚀 Deploying Cloud Function: $FUNCTION_NAME"

gcloud functions deploy "$FUNCTION_NAME" \
  --gen2 \
  --region=us-central1 \
  --runtime=go122 \
  --source="$SOURCE_DIR" \
  --entry-point="$ENTRY_POINT" \
  --trigger-http \
  --allow-unauthenticated

echo "✅ Deployment complete!"

