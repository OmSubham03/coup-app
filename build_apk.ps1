$env:JAVA_HOME = "C:\Program Files\Android\openjdk\jdk-21.0.8"
$env:ANDROID_HOME = "$env:LOCALAPPDATA\Android\Sdk"

Write-Host "Building APK..." -ForegroundColor Cyan
Push-Location "$PSScriptRoot\mobile\android"
.\gradlew.bat assembleDebug 2>&1 | Out-Null
if ($LASTEXITCODE -ne 0) {
    .\gradlew.bat assembleDebug
    Pop-Location
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}
Pop-Location

$apk = "$PSScriptRoot\mobile\android\app\build\outputs\apk\debug\app-debug.apk"
$dest = "$PSScriptRoot\CoupGames.apk"
Copy-Item $apk $dest -Force
$size = [math]::Round((Get-Item $dest).Length / 1MB, 1)
Write-Host "APK built: CoupGames.apk ($size MB)" -ForegroundColor Green
