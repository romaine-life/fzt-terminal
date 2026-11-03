# Publish the latest Forgejo release's Windows `fzt` to \\NelsonNAS\tools as `fzt`
# (Nelson's home machines run it from there; see shell-config-profile-home
# claude\topics\home-tools.md). Cut a release first (release.yml).
$ErrorActionPreference = 'Stop'

$repo = 'romaine-life/fzt-terminal'
$api = "https://forgejo.romaine.life/api/v1/repos/$repo"
$cred = "protocol=https`nhost=forgejo.romaine.life`n`n" | git credential fill
$token = ($cred | Where-Object { $_ -like 'password=*' }) -replace '^password=', ''
if (-not $token) { throw 'no forgejo.romaine.life token in the git credential helper' }
$h = @{ Authorization = "token $token" }

$release = Invoke-RestMethod "$api/releases/latest" -Headers $h
$tag = $release.tag_name
$commit = (Invoke-RestMethod "$api/tags/$tag" -Headers $h).commit.sha
$asset = $release.assets | Where-Object name -eq 'fzt-windows-amd64.exe'
if (-not $asset) { throw "release $tag has no fzt-windows-amd64.exe" }

$dir = Join-Path ([IO.Path]::GetTempPath()) "fzt-$tag-$([Guid]::NewGuid().ToString('N'))"
New-Item -ItemType Directory $dir | Out-Null
Invoke-WebRequest $asset.browser_download_url -Headers $h -OutFile (Join-Path $dir 'fzt.exe')
$signature = Get-AuthenticodeSignature (Join-Path $dir 'fzt.exe')
if ($signature.Status -ne 'Valid') { throw "release $tag has an invalid signature: $($signature.Status)" }

publish-tool -Name fzt -Version $tag -Path (Join-Path $dir 'fzt.exe') `
    -Entry fzt.exe -Repo "https://forgejo.romaine.life/$repo" -Commit $commit
