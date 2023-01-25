# Working old command prompt script

# @echo off
# mkdir %1
# cd %1
# echo package main> %1.go
# go mod init %1
# cd ..
# go work use %1

$file_name = $args[0]
if (Test-Path $file_name) {
    Write-Host "Folder already exist"
    exit
}
New-Item -ItemType Directory -Path $file_name
Set-Location $file_name
New-Item -ItemType File -Path . -Name "$file_name.go"
Set-Content -Path "$file_name.go" -Value "package main"
go mod init $file_name
Set-Location ..