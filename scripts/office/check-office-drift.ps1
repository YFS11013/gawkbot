#requires -Version 7
<#
.SYNOPSIS
  office 漂移检查：把 broker 的实时调度状态与 docs/office/expected-scheduler.json 对照，只输出越界项。

.DESCRIPTION
  检查项（任一不满足即 exit 1）：
    1. 期望 job 是否存在
    2. enabled / schedule_expr / interval_minutes / target_id / channel 是否与期望一致
    3. requires_file 指向的脚本或 workflow 定义是否真的存在（本机事故：workflow 定义从未生成，每天静默 failed）
    4. 期望启用的 job 的 last_run_status 是否为 ok/空
    5. 期望启用的 workflow job，其 skill 的 last_execution_status 是否 failed
    6. 出现清单外的新 standing job（per-task follow-up / recheck 类 churn job 不比对）
    7. scheduled task 的脚本是否存在，日志是否新鲜（max_age_hours）

  权威仍是运行时：本脚本只读、只报，不 apply。要改期望态就改 JSON 并提交，然后按 RUNTIME-API.md 走 HTTP。

  告警（可选）：传 -AlertsPath + -StatePath 时，违规集按内容哈希去重后追加到 alerts 文件，
  恢复时输出 `DRIFT: cleared` 并删除状态文件。首行永远是 ASCII 的 `DRIFT: none|cleared|<n>`，
  供 Windows PowerShell 5.1 的调用方（watchdog）解析 —— 5.1 解码子进程中文会变乱码，故勿依赖中文行。

.EXAMPLE
  pwsh -File scripts/office/check-office-drift.ps1
#>
param(
    [string]$ManifestPath = (Join-Path (Split-Path -Parent (Split-Path -Parent $PSScriptRoot)) 'docs\office\expected-scheduler.json'),
    [string]$AlertsPath,
    [string]$StatePath
)
$ErrorActionPreference = 'Stop'
$RepoRoot   = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$TokenFile  = Join-Path $env:TEMP 'wuphf-broker-token-7892'
$BrokerUrl  = 'http://127.0.0.1:7892'
$TokenUrl   = 'http://127.0.0.1:7891/api-token'

# churn job：由任务/请求生命周期自动增删，不在期望清单里比对，也不算“新增 standing job”
$ChurnPattern = '^(task-follow-up|request-follow-up|recheck|task-reminder|request-reminder):'

function Get-BrokerToken {
    if (Test-Path -LiteralPath $TokenFile) {
        $t = (Get-Content -LiteralPath $TokenFile -Raw).Trim()
        if ($t) { return $t }
    }
    return (Invoke-RestMethod -Uri $TokenUrl -TimeoutSec 15).token
}

$violations = [System.Collections.Generic.List[string]]::new()
function Add-Violation([string]$msg) { $violations.Add($msg) }

if (-not (Test-Path -LiteralPath $ManifestPath)) { throw "期望态清单缺失: $ManifestPath" }
$manifest = Get-Content -LiteralPath $ManifestPath -Raw | ConvertFrom-Json

$headers = @{ Authorization = 'Bearer ' + (Get-BrokerToken) }
$live     = (Invoke-RestMethod -Uri "$BrokerUrl/scheduler" -Headers $headers -TimeoutSec 20).jobs
$skills   = (Invoke-RestMethod -Uri "$BrokerUrl/skills"    -Headers $headers -TimeoutSec 20).skills

$liveBySlug = @{}
foreach ($j in $live) { $liveBySlug[$j.slug] = $j }

$expectedSlugs = @{}
foreach ($e in $manifest.broker.expected_jobs) {
    $expectedSlugs[$e.slug] = $true
    $job = $liveBySlug[$e.slug]
    if (-not $job) { Add-Violation "缺失 job: $($e.slug)（期望 enabled=$($e.enabled)）"; continue }

    if ($job.enabled -ne $e.enabled) {
        Add-Violation "enabled 漂移: $($e.slug) 期望=$($e.enabled) 实际=$($job.enabled)"
    }
    foreach ($field in 'schedule_expr', 'target_id', 'channel') {
        if ($null -ne $e.$field -and "$($job.$field)" -ne "$($e.$field)") {
            Add-Violation "$field 漂移: $($e.slug) 期望=$($e.$field) 实际=$($job.$field)"
        }
    }
    if ($null -ne $e.interval_minutes -and [int]$job.interval_minutes -ne [int]$e.interval_minutes) {
        Add-Violation "interval_minutes 漂移: $($e.slug) 期望=$($e.interval_minutes) 实际=$($job.interval_minutes)"
    }
    if ($e.enabled -and $e.requires_file -and -not (Test-Path -LiteralPath $e.requires_file)) {
        Add-Violation "依赖文件缺失: $($e.slug) → $($e.requires_file)（该 job 每次运行都会失败）"
    }
    if ($e.enabled -and $job.last_run_status -and $job.last_run_status -ne 'ok') {
        Add-Violation "最近一次运行失败: $($e.slug) last_run_status=$($job.last_run_status)"
    }
    if ($e.enabled -and $e.skill) {
        $sk = $skills | Where-Object { $_.name -eq $e.skill } | Select-Object -First 1
        if (-not $sk) {
            Add-Violation "skill 不存在: $($e.slug) → $($e.skill)"
        } elseif ($sk.last_execution_status -eq 'failed') {
            Add-Violation "skill 执行失败: $($e.skill) last_execution_status=failed（静默失败就在这一行）"
        }
    }
}

foreach ($j in $live) {
    if ($j.slug -match $ChurnPattern) { continue }
    if (-not $expectedSlugs.ContainsKey($j.slug)) {
        Add-Violation "清单外 job: $($j.slug)（enabled=$($j.enabled)）—— 加进期望态或停用，别让它无人看管地跑"
    }
}

foreach ($t in $manifest.scheduled_tasks) {
    if ($t.requires_file -and -not (Test-Path -LiteralPath $t.requires_file)) {
        Add-Violation "计划任务脚本缺失: $($t.name) → $($t.requires_file)"
        continue
    }
    if ($t.max_age_hours -and $t.log) {
        if (-not (Test-Path -LiteralPath $t.log)) {
            Add-Violation "计划任务日志缺失: $($t.name) → $($t.log)"
        } else {
            $age = ((Get-Date).ToUniversalTime() - (Get-Item -LiteralPath $t.log).LastWriteTimeUtc).TotalHours
            if ($age -gt [double]$t.max_age_hours) {
                Add-Violation ("计划任务停更: {0} 日志 {1:N1}h 未更新（阈值 {2}h）" -f $t.name, $age, $t.max_age_hours)
            }
        }
    }
}

if ($violations.Count -eq 0) {
    if ($StatePath -and (Test-Path -LiteralPath $StatePath)) {
        Remove-Item -LiteralPath $StatePath -Force
        Write-Output 'DRIFT: cleared'
    } else {
        Write-Output 'DRIFT: none'
    }
    Write-Output '期望态与运行时一致'
    exit 0
}

Write-Output "DRIFT: $($violations.Count)"
$violations | ForEach-Object { Write-Output "  - $_" }

# Alerting lives HERE (PowerShell 7) and not in the caller: the watchdog runs
# under Windows PowerShell 5.1, where Chinese text crossing a process boundary
# gets decoded with the OEM code page and arrives as mojibake. The dedupe key is
# the hash of the violation set, so the same fault alerts once, not every run.
if ($AlertsPath -and $StatePath) {
    $joined = ($violations -join "`n")
    $sha = [System.Security.Cryptography.SHA256]::Create()
    try {
        $hash = [System.BitConverter]::ToString(
            $sha.ComputeHash([System.Text.Encoding]::UTF8.GetBytes($joined))).Replace('-', '')
    } finally {
        $sha.Dispose()
    }
    $prev = ''
    if (Test-Path -LiteralPath $StatePath) { $prev = (Get-Content -LiteralPath $StatePath -Raw).Trim() }
    if ($prev -ne $hash) {
        $dir = Split-Path -Parent $AlertsPath
        if ($dir -and -not (Test-Path -LiteralPath $dir)) { New-Item -ItemType Directory -Force -Path $dir | Out-Null }
        $stamp = (Get-Date).ToUniversalTime().ToString('yyyy-MM-dd HH:mm UTC')
        Add-Content -LiteralPath $AlertsPath -Encoding utf8 -Value "`n## $stamp office drift`n`n$($violations -join "`n")`n"
        Set-Content -LiteralPath $StatePath -Value $hash -Encoding utf8
        Write-Output "ALERT: appended to $AlertsPath"
    } else {
        Write-Output 'ALERT: unchanged since last run (deduped)'
    }
}

exit 1
