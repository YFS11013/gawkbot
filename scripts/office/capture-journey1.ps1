# capture-journey1.ps1 — snapshot the evidence for the Journey 1 run.
#
# Usage:
#   pwsh -NoProfile -File scripts/office/capture-journey1.ps1 -Out docs/office/core-loop-runs/2026-11-07
#
# Writes, under -Out/snapshots/<HHmmss>/:
#   tasks.json      every non-archived task with owner / lifecycle / parent link
#   activity-<id>.json, messages-<channel>.json for the tasks in scope
#   wiki-log.txt    git log of the workspace wiki
#   wiki-tree.txt   file list with sizes and mtimes
#   health.json     provider / memory backend / member count
#
# The workspace name and ports are arguments because a second isolated office
# (Demo Day) reuses the same script with different values.

param(
    [string]$Workspace = 'journey1',
    [int]$BrokerPort = 7910,
    [string]$Out = 'docs/office/core-loop-runs/2026-11-07',
    [string[]]$TaskIds = @('JOURN-22', 'JOURN-24')
)

$ErrorActionPreference = 'Stop'
$space = Join-Path $env:USERPROFILE ".wuphf-spaces\$Workspace"
$tokenPath = Join-Path $env:TEMP "wuphf-broker-token-$BrokerPort"
$tok = (Get-Content -LiteralPath $tokenPath -Raw).Trim()
$headers = @{ Authorization = "Bearer $tok" }
$base = "http://127.0.0.1:$BrokerPort"

$stamp = Get-Date -Format 'HHmmss'
$dir = Join-Path $Out "snapshots\$stamp"
New-Item -ItemType Directory -Path $dir -Force | Out-Null

# Tasks: read the broker's own state file so archived/canceled history is not
# hidden by an API default filter, then project the same fields the API serves.
$state = Get-Content -LiteralPath (Join-Path $space '.wuphf\team\broker-state.json') -Raw | ConvertFrom-Json
$tasks = @($state.tasks | Where-Object { $_.lifecycle_state -notin @('archived') } |
    ForEach-Object {
        [pscustomobject]@{
            id              = $_.id
            title           = $_.title
            status          = $_.status
            lifecycle_state = $_.lifecycle_state
            owner           = $_.owner
            created_by      = $_.created_by
            created_at      = $_.created_at
            updated_at      = $_.updated_at
            parent_issue_id = $_.parent_issue_id
            pipeline_stage  = $_.pipeline_stage
            review_state    = $_.review_state
            channel         = $_.channel
        }
    })
$tasks | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath (Join-Path $dir 'tasks.json') -Encoding utf8

foreach ($id in $TaskIds) {
    try {
        (Invoke-RestMethod -Uri "$base/tasks/$id/activity" -Headers $headers -TimeoutSec 20) |
            ConvertTo-Json -Depth 8 | Set-Content -LiteralPath (Join-Path $dir "activity-$id.json") -Encoding utf8
        (Invoke-RestMethod -Uri "$base/tasks/$id" -Headers $headers -TimeoutSec 20) |
            ConvertTo-Json -Depth 8 | Set-Content -LiteralPath (Join-Path $dir "task-$id.json") -Encoding utf8
    } catch {
        "capture: $id failed: $($_.Exception.Message)" | Set-Content -LiteralPath (Join-Path $dir "error-$id.txt") -Encoding utf8
    }
}

foreach ($ch in @('cos__human')) {
    try {
        (Invoke-RestMethod -Uri "$base/messages?channel=$ch&limit=60" -Headers $headers -TimeoutSec 20) |
            ConvertTo-Json -Depth 8 | Set-Content -LiteralPath (Join-Path $dir "messages-$ch.json") -Encoding utf8
    } catch {
        "capture: channel $ch failed: $($_.Exception.Message)" | Set-Content -LiteralPath (Join-Path $dir "error-$ch.txt") -Encoding utf8
    }
}

$wiki = Join-Path $space '.wuphf\wiki'
if (Test-Path $wiki) {
    git -C $wiki log --oneline -40 | Out-File -FilePath (Join-Path $dir 'wiki-log.txt') -Encoding utf8
    git -C $wiki ls-files -z | ForEach-Object { $_ -split "`0" } | Where-Object { $_ } | ForEach-Object {
        $full = Join-Path $wiki $_
        if (Test-Path -LiteralPath $full) {
            $f = Get-Item -LiteralPath $full
            '{0}`t{1}`t{2}' -f $f.Length, $f.LastWriteTime.ToString('yyyy-MM-dd HH:mm:ss'), $_
        }
    } | Set-Content -LiteralPath (Join-Path $dir 'wiki-tree.txt') -Encoding utf8
}

(Invoke-RestMethod -Uri "$base/health" -TimeoutSec 20) |
    ConvertTo-Json -Depth 6 | Set-Content -LiteralPath (Join-Path $dir 'health.json') -Encoding utf8

"captured -> $dir"
