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
    '(^|/)\.DS_Store$',
    '(^|/)Thumbs\.db$',
    '\.log$',
    '\.local$',
    '\.pem$',
    '\.key$',
    '\.ppk$',
    '\.p12$',
    '\.pfx$',
    '\.iml$',
    '\.swp$'
)

$privateKeyMarkerPattern = [regex]'-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----'
$privateKeyScanTailLength = 128

function ConvertTo-ArchivePath {
    param([Parameter(Mandatory = $true)][string]$Path)

    $archivePath = $Path -replace '\\', '/'
    while ($archivePath.StartsWith("./")) {
        $archivePath = $archivePath.Substring(2)
    }
    while ($archivePath.StartsWith("/")) {
        $archivePath = $archivePath.Substring(1)
    }
    return $archivePath
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

            $stream = $entry.Open()
            try {
                $reader = [System.IO.StreamReader]::new($stream, [System.Text.Encoding]::UTF8, $true, 65536)
                try {
                    $buffer = [char[]]::new(65536)
                    $tail = ""

                    while (($read = $reader.Read($buffer, 0, $buffer.Length)) -gt 0) {
                        $chunk = [string]::new($buffer, 0, $read)
                        $window = $tail + $chunk
                        if ($privateKeyMarkerPattern.IsMatch($window)) {
                            throw "Refusing to export archive containing private key marker: $($entry.FullName)"
                        }

                        if ($window.Length -gt $privateKeyScanTailLength) {
                            $tail = $window.Substring($window.Length - $privateKeyScanTailLength)
                        }
                        else {
                            $tail = $window
                        }
                    }
                }
                finally {
                    $reader.Dispose()
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

function Publish-ValidatedArchive {
    param(
        [Parameter(Mandatory = $true)]
        [string]$TempPath,

        [Parameter(Mandatory = $true)]
        [string]$OutputPath,

        [switch]$Force
    )

    if ((Test-Path $OutputPath) -and -not $Force) {
        throw "Archive already exists: $OutputPath. Pass -Force to overwrite it."
    }

    if (Test-Path $OutputPath) {
        $backupPath = "{0}.backup-{1}" -f $OutputPath, ([guid]::NewGuid().ToString("N"))
        try {
            [System.IO.File]::Replace($TempPath, $OutputPath, $backupPath)
        }
        finally {
            if (Test-Path $backupPath) {
                Remove-Item $backupPath -Force
            }
        }
        return
    }

    [System.IO.File]::Move($TempPath, $OutputPath)
}

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$outputFullPath = [System.IO.Path]::GetFullPath((Join-Path $repoRoot $OutputPath))
$outputDir = Split-Path -Parent $outputFullPath
$tempFileName = "{0}.tmp-{1}" -f ([System.IO.Path]::GetFileName($outputFullPath)), ([guid]::NewGuid().ToString("N"))
$tempFullPath = Join-Path $outputDir $tempFileName
$archivePublished = $false

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
    Invoke-Native "git" @("archive", "--format=zip", "--output", $tempFullPath, $Treeish)

    $archive = [System.IO.Compression.ZipFile]::OpenRead($tempFullPath)
    try {
        $entryPaths = @($archive.Entries | ForEach-Object { $_.FullName })
    }
    finally {
        $archive.Dispose()
    }

    Assert-NoBlockedArchivePath -Paths $entryPaths -Source $tempFullPath
    Assert-NoPrivateKeyMarker -ArchivePath $tempFullPath

    Publish-ValidatedArchive -TempPath $tempFullPath -OutputPath $outputFullPath -Force:$Force
    $archivePublished = $true

    Write-Host "Archive created: $outputFullPath"
}
finally {
    if ((-not $archivePublished) -and (Test-Path $tempFullPath)) {
        Remove-Item $tempFullPath -Force
    }
    Pop-Location
}
