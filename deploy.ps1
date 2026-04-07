$registry = "coupgamesacr"
$app = "coup-server"
$rg = "coup-rg"
$tag = "coup-server:" + (Get-Date -Format "yyyyMMddHHmmss")

Write-Host "Attempting Az Login"
az account set --subscription "Visual Studio Enterprise Subscription"
Write-Host "Building and pushing image ($tag)..." -ForegroundColor Cyan
az acr build --registry $registry --image $tag --file server/Dockerfile server/
if ($LASTEXITCODE -ne 0) { Write-Host "Build failed!" -ForegroundColor Red; exit 1 }

Write-Host "Updating container app..." -ForegroundColor Cyan
az containerapp update --name $app --resource-group $rg --image "$registry.azurecr.io/$tag" | Out-Null
if ($LASTEXITCODE -ne 0) { Write-Host "Update failed!" -ForegroundColor Red; exit 1 }

Write-Host "Deployed successfully!" -ForegroundColor Green
