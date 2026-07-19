param(
    [string]$OutputPath = ".artifacts/pet-study-source.zip",
    [string]$Treeish = "HEAD",
    [switch]$Force
)

$ErrorActionPreference = "Stop"
. "$PSScriptRoot\\common.ps1"
Add-Type -AssemblyName System.IO.Compression.FileSystem

$blockedPathPatterns = @(
    '(^|/)\.git(/|$)',
    '(^|/)\.idea(/|$)',
    '(^|/)\.vscode(/|$)',
    '(^|/)\.artifacts(/|$)',
    '(^|/)certs(/|$)',
    '(^|/)bin(/|$)',
    '(^|/)dist(/|$)',
    '(^|/)tmp(/|$)',
    '(^|/)-H$',
    '(^|/)req(?:-[^/]+)?\.json$',
    '(^|/)\.env(?:\..*)?$',
    '\.log$',
    '\.local$',
    '\.pem$',
    '\.key$'
)

function ConvertTo-ArchivePath {
    param([Parameter(Mandatory = $true)][string]$Path)

    return ($Path -replace '\\', '/').TrimStart('./')
}

function Assert-NoBlockedArchivePath {
    param(
        [Parameter(Mandatory = $true)]
        [string[]]$Paths,

        [Parameter(Mandatory = $true)]
        [string]$Source
    )

    foreach ($path in $Paths) {
        $archivePath = ConvertTo-ArchivePath $path
        foreach ($pattern in $blockedPathPatterns) {
            if ($archivePath -match $pattern) {
                throw "Refusing to export $Source because blocked path matched ${pattern}: $archivePath"
            }
        }
    }
}

function Assert-NoPrivateKeyMarker {
    param([Parameter(Mandatory = $true)][string]$ArchivePath)

    $archive = [System.IO.Compression.ZipFile]::OpenRead($ArchivePath)
    try {
        foreach ($entry in $archive.Entries) {
            if ($entry.FullName.EndsWith('/')) {
                continue
            }
            if ($entry.Length -gt 1MB) {
                continue
            }

            $stream = $entry.Open()
            try {
                $reader = [System.IO.StreamReader]::new($stream)
                $content = $reader.ReadToEnd()
                if ($content -match '-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----') {
                    throw "Refusing to export archive containing private key marker: $($entry.FullName)"
                }
            }
            finally {
                $stream.Dispose()
            }
        }
    }
    finally {
        $archive.Dispose()
    }
}

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$outputFullPath = [System.IO.Path]::GetFullPath((Join-Path $repoRoot $OutputPath))
$outputDir = Split-Path -Parent $outputFullPath

Push-Location $repoRoot
try {
    $trackedPaths = git ls-tree -r --name-only $Treeish
    if ($LASTEXITCODE -ne 0) {
        throw "Command failed with exit code ${LASTEXITCODE}: git ls-tree -r --name-only $Treeish"
    }
    Assert-NoBlockedArchivePath -Paths $trackedPaths -Source $Treeish

    if ((Test-Path $outputFullPath) -and -not $Force) {
        throw "Archive already exists: $outputFullPath. Pass -Force to overwrite it."
    }

    New-Item -ItemType Directory -Force -Path $outputDir | Out-Null
    Invoke-Native "git" @("archive", "--format=zip", "--output", $outputFullPath, $Treeish)

    $archive = [System.IO.Compression.ZipFile]::OpenRead($outputFullPath)
    try {
        $entryPaths = @($archive.Entries | ForEach-Object { $_.FullName })
    }
    finally {
        $archive.Dispose()
    }

    Assert-NoBlockedArchivePath -Paths $entryPaths -Source $outputFullPath
    Assert-NoPrivateKeyMarker -ArchivePath $outputFullPath

    Write-Host "Archive created: $outputFullPath"
}
finally {
    Pop-Location
}
