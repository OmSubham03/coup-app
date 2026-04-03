$registry = "cae427b78968acr"
$image = "coup-server:latest"
$app = "coup-server"
$rg = "coup-rg"

Write-Host "Building and pushing image..." -ForegroundColor Cyan
az acr build --registry $registry --image $image --file server/Dockerfile server/
if ($LASTEXITCODE -ne 0) { Write-Host "Build failed!" -ForegroundColor Red; exit 1 }

Write-Host "Updating container app..." -ForegroundColor Cyan
az containerapp update --name $app --resource-group $rg --image "$registry.azurecr.io/$image" | Out-Null
if ($LASTEXITCODE -ne 0) { Write-Host "Update failed!" -ForegroundColor Red; exit 1 }

Write-Host "Deployed successfully!" -ForegroundColor Green
