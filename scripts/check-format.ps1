param()

$ErrorActionPreference = "Stop"

$files = gofmt -l cmd internal scripts
if ($files) {
    Write-Error "gofmt required for:`n$files"
}
