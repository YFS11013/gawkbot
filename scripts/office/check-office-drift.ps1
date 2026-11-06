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
    7. scheduled task 的脚本是否存在，日志是否新鲜（max_age_hours）。
       enabled=false 的任务整条跳过：停用的任务日志本来就不该再更新，对它查新鲜度
       只会产生一条永远为真的告警。
    8. 监听 7892 的进程跑的是不是当前代码 —— 两个方向都要查：
         · 进程启动时间 < 二进制 mtime  → 重建了没重启
         · 二进制 mtime < 最近的 Go 提交 → 改了没重建
       2 和 3 常被当成同一件事。2026-11-02：修复提交、测试全绿、二进制重建，
       而 broker 又继续用旧构建服务了几个小时，因为没有任何东西比对过这两者。

  权威仍是运行时：本脚本只读、只报，不 apply。要改期望态就改 JSON 并提交，然后按 RUNTIME-API.md 走 HTTP。

  告警（可选）：传 -AlertsPath + -StatePath 时，违规集按**规范化后**的内容哈希去重后追加到 alerts 文件，
  恢复时输出 `DRIFT: cleared` 并删除状态文件。规范化（ConvertTo-DedupeKey）剔除了会自己变的数字
  —— 年龄、超时小时数、提交列表、时间戳、临时目录名 —— 否则同一个长期故障每跑一次都会算成新故障
  （2026-11-03..06：一条故障写了 301 条告警）。首行永远是 ASCII 的 `DRIFT: none|cleared|<n>`，
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

# The running broker may be built under any file name (gawkbot-fix.exe,
# wuphf.exe, ...), so ask the listening port which process owns it instead of
# guessing. Returns the executable path, or $null when nothing is listening —
# which is itself drift worth reporting: the scheduler checks below cannot mean
# anything if no broker is serving them.
function Get-RunningBrokerBinary {
    $conn = Get-NetTCPConnection -LocalPort 7892 -State Listen -ErrorAction SilentlyContinue |
        Select-Object -First 1
    if (-not $conn) { return $null }
    $proc = Get-CimInstance Win32_Process -Filter "ProcessId=$($conn.OwningProcess)" -ErrorAction SilentlyContinue
    if (-not $proc -or -not $proc.ExecutablePath) { return $null }
    return [pscustomobject]@{
        Path       = $proc.ExecutablePath
        StartedUtc = $proc.CreationDate.ToUniversalTime()
    }
}

# A green test run says the code is correct; it says nothing about whether that
# code is what is answering requests. On 2026-11-02 a fix landed, tests passed,
# the binary was rebuilt — and the broker went on serving the previous build for
# hours, because nothing compared the two.
#
# Two separate gaps, and checking only one of them lies:
#   rebuilt but not restarted  -> process start < binary mtime
#   restarted but not rebuilt  -> binary mtime  < last Go commit
# The first is the one an operator actually forgets, and it is invisible to any
# check that looks only at files on disk.
function Get-DeployLag($run, [string]$repoRoot) {
    $notes = @()
    if (-not $run) { return @('找不到监听 7892 的 broker 进程——无法确认被测版本就是运行版本') }
    $info = Get-Item -LiteralPath $run.Path -ErrorAction SilentlyContinue
    if (-not $info) { return @("运行中的可执行文件读不到: $($run.Path)") }

    if ($run.StartedUtc -and $run.StartedUtc -lt $info.LastWriteTimeUtc) {
        $notes += ("进程跑的是旧代码: {0} 启动于 {1:yyyy-MM-dd HH:mm} UTC，但该文件后来在 {2:yyyy-MM-dd HH:mm} UTC 被重新构建过——重建了没重启" -f `
                (Split-Path -Leaf $run.Path), $run.StartedUtc, $info.LastWriteTimeUtc)
    }

    # The trailing Z is load-bearing. git parses a bare ISO 8601 date as LOCAL
    # time, while LastWriteTimeUtc is UTC; on this machine that is an 8-hour
    # skew, which silently swept in the whole day's commits and made a
    # just-rebuilt binary look stale.
    $since = $info.LastWriteTimeUtc.ToString('yyyy-MM-ddTHH:mm:ss') + 'Z'
    # Exclude _test.go: a commit that only touches tests cannot change what the
    # running binary does, so it must not read as an undeployed fix. Without
    # this, a green test run "proves" nothing is stale while the script insists
    # the build is behind -- and the honest fix is not to rebuild, it is to ask
    # the right question.
    $hashes = & git -C $repoRoot log "--since=$since" --format=%h -- '*.go' ':!*_test.go' 2>$null
    if ($LASTEXITCODE -eq 0) {
        $behind = @($hashes | Where-Object { $_ })
        if ($behind.Count -gt 0) {
            $recent = ($behind | Select-Object -First 5) -join ', '
            $more = if ($behind.Count -gt 5) { ', …' } else { '' }
            $notes += ("运行中的二进制落后于代码: 构建于 {0:yyyy-MM-dd HH:mm} UTC，之后有 {1} 个 Go 提交（{2}{3}）未部署——测试绿不代表这段代码在跑" -f `
                    $info.LastWriteTimeUtc, $behind.Count, $recent, $more)
        }
    }
    return $notes
}
# ConvertTo-DedupeKey turns a violation line into the identity used for dedupe.
#
# A hash of the violation text is not a dedupe key when the text embeds numbers
# that change on their own: `日志 26.1h 未更新` and `日志 100.1h 未更新` are the
# same fault but hash differently. On 2026-11-03..06 that made one chronic fault
# append 301 sections to wiki/team/alerts.md, 15 minutes apart, because the
# reported age grew every run and the state hash never matched.
#
# What stays: which thing failed and how (task/job name, field, kind). What
# goes: durations, hours-overdue, counts, timestamps, commit hashes, and any
# path with a temp-directory segment (temp dirs get a fresh name every process,
# so the same fault would otherwise look new every run).
#
# Deliberately NOT stripping names like broker.err.log: the file is part of the
# fault's identity, and that name does not change between runs.
function ConvertTo-DedupeKey([string]$msg) {
    $s = $msg
    # Order matters: dates and hashes must be masked BEFORE the bare-number pass,
    # or that pass eats their digits first and the pattern never matches.
    # ① UTC timestamps and bare dates
    $s = [regex]::Replace($s, '\d{4}-\d{2}-\d{2}(?:[T ]\d{2}:\d{2}(?::\d{2})?)?', '<ts>')
    # ② commit short hashes; a commit LIST is also one identity — the number of
    #    undeployed commits grows as work lands, but the fault is unchanged.
    $s = [regex]::Replace($s, '\b[0-9a-f]{7,40}\b', '<sha>')
    $s = [regex]::Replace($s, '<sha>(?:, <sha>)*(?:, …)?', '<sha>')
    # ③ explicit durations (26.1h / 90m) and the parenthesized threshold form
    $s = [regex]::Replace($s, '\b\d+(?:\.\d+)?\s*(?:h|m|s)\b', '<dur>')
    $s = [regex]::Replace($s, '\(\d+(?:\.\d+)?\)', '(N)')
    # ④ remaining bare numbers are counts ("3 个 Go 提交"). The lookbehind keeps
    #    file extensions intact: thesis-week.ps1 must not become thesis-week.ps<n>.
    #    The +N form is the binary-build stamp ("…+2026-11-02T08:37") whose time
    #    advances whenever the file is touched, even when the fault is unchanged.
    $s = [regex]::Replace($s, '\+\d+', '<n>')
    $s = [regex]::Replace($s, '(?<![A-Za-z0-9])\d+(?:\.\d+)?', '<n>')
    # ⑤ paths under a temp dir: the directory name is unique per process
    $s = [regex]::Replace($s, '(?i)[A-Za-z]:[\\/][^\s]*[\\/]temp[\\/][^\s]+', '<tmp>')
    $s = [regex]::Replace($s, '(?i)[\\/](?:tmp|Temp)[\\/][^\s]+', '<tmp>')
    # ⑥ collapse whitespace so formatting noise cannot split one key into two
    return ([regex]::Replace($s, '\s+', ' ')).Trim()
}

$violations = [System.Collections.Generic.List[string]]::new()
# $violationKeys mirrors $violations, but with the numbers that drift every run
# stripped out — that is what the state hash is taken over.
$violationKeys = [System.Collections.Generic.List[string]]::new()
function Add-Violation([string]$msg) {
    $violations.Add($msg)
    $violationKeys.Add((ConvertTo-DedupeKey $msg))
}

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

# 9. enabled 的 job，next_run 是否已经过期。
#
# last_run_status 和 next_run 各自都能撒谎，合起来才是证词。2026-11-02 实测：
# 四个 routine 的 next_run 停在 2026-10-29/30，而 last_run_status 一直是 ok ——
# 每个字段单独看都"正常"，合起来是"一周没动过"。整张排班表 18 行看起来健康，
# 实际只有一个业务 routine 在 fire。
#
# churn job 不算：它们的 next_run 由任务生命周期驱动，任务进终态就该消失。
# 没有 next_run 的也不查：系统 cron 由 broker 自己的 goroutine 驱动，不走这个字段。
$nowUtc = (Get-Date).ToUniversalTime()
foreach ($j in $live) {
    if ($j.slug -match $ChurnPattern) { continue }
    if ($j.enabled -eq $false) { continue }
    if (-not $j.next_run) { continue }
    $nextUtc = [datetime]::MinValue
    $parsed = [datetime]::TryParse(
        [string]$j.next_run,
        [System.Globalization.CultureInfo]::InvariantCulture,
        [System.Globalization.DateTimeStyles]::AdjustToUniversal -bor [System.Globalization.DateTimeStyles]::AssumeUniversal,
        [ref]$nextUtc)
    if (-not $parsed) {
        Add-Violation "next_run 无法解析: $($j.slug) → $($j.next_run)"
        continue
    }
    $overdueHours = ($nowUtc - $nextUtc).TotalHours
    if ($overdueHours -gt 2) {
        Add-Violation ("next_run 已过期 {0:N1}h: {1}（last_run={2} last_run_status={3}）—— 排班表上活着，实际没在跑" -f `
            $overdueHours, $j.slug, $j.last_run, $j.last_run_status)
    }
}

foreach ($t in $manifest.scheduled_tasks) {
    # enabled=false 是期望态的一部分（2026-11-02 最小配置试验停用了三个任务）。
    # 停用的任务日志当然不新鲜：不加这道门，`max_age_hours` 会对一个"有意不再更新"
    # 的日志永远报警，且报警文本里嵌着不断增长的年龄数字（26.1h → 100.1h），
    # 内容哈希每分钟都不同 —— 去重永不命中，alerts.md 每 15 分钟长一条。
    # 2026-11-06 实测：301 条重复告警，把 wiki 里 33k 字符的日志喂给 30 分钟一次的
    # skill-compile cron，单篇判定 input 从 15k 涨到 28.6k，占当天 skillify 消耗的 99%。
    if ($t.enabled -eq $false) { continue }
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

# 8. 运行中的二进制是否落后于代码。
#    这一项与调度无关，但它让上面 1-7 的绿有意义：所有检查都通过，可能只是
#    因为跑的还是那个早就通过了检查的旧版本。
foreach ($note in (Get-DeployLag (Get-RunningBrokerBinary) $RepoRoot)) {
    Add-Violation $note
}

if ($violations.Count -eq 0) {
    if ($StatePath -and (Test-Path -LiteralPath $StatePath)) {
        Remove-Item -LiteralPath $StatePath -Force
        Write-Output 'DRIFT: cleared'
    } else {
        Write-Output 'DRIFT: none'
    }
    # Coverage, not just a verdict. "DRIFT: none" answers "did anything I looked
    # at fail" and stays silent on "what did I look at" -- which is how four
    # broker routines sat at next_run = 2026-10-29 for a week with
    # last_run_status = ok, and how the 2026-11-02 count looked healthy at 18
    # jobs while one business routine actually fired. A checker that cannot
    # report its own scope cannot distinguish "clean" from "barely looked".
    #
    # The first line stays exactly `DRIFT: none|cleared|<n>` because the
    # watchdog parses it; the counts ride on the second line.
    $jobCount   = @($manifest.broker.expected_jobs).Count
    $taskCount  = @($manifest.scheduled_tasks).Count
    $binarySeen = if (Get-RunningBrokerBinary) { 1 } else { 0 }
    Write-Output ('期望态与运行时一致（比对 {0} 个 job、{1} 个计划任务、{2} 个运行中二进制）' -f $jobCount, $taskCount, $binarySeen)
    exit 0
}

Write-Output "DRIFT: $($violations.Count)"
$violations | ForEach-Object { Write-Output "  - $_" }

# Alerting lives HERE (PowerShell 7) and not in the caller: the watchdog runs
# under Windows PowerShell 5.1, where Chinese text crossing a process boundary
# gets decoded with the OEM code page and arrives as mojibake. The dedupe key is
# the hash of the NORMALIZED violation set (see ConvertTo-DedupeKey), so the same
# fault alerts once even when its reported magnitude keeps changing; a new fault
# still changes the key set and still alerts.
if ($AlertsPath -and $StatePath) {
    # Sort so the key set is order-independent: a scan that returns the same
    # faults in a different order is the same state, not a new one. Ordinal
    # (not culture) because the hash must not change when the host's UI culture
    # does -- otherwise a locale change re-alerts every standing fault once.
    $sortedKeys = [System.Collections.Generic.List[string]]::new($violationKeys)
    $sortedKeys.Sort([System.StringComparer]::Ordinal)
    $keySet = $sortedKeys -join "`n"
    $sha = [System.Security.Cryptography.SHA256]::Create()
    try {
        $hash = [System.BitConverter]::ToString(
            $sha.ComputeHash([System.Text.Encoding]::UTF8.GetBytes($keySet))).Replace('-', '')
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
