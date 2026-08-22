<#
.SYNOPSIS
    Blender Remote Console (BRC) CLI tool.
.DESCRIPTION
    CLI helper for Blender Remote Console addon. Analogous to adb commands.
#>

# Parse Arguments Manually for 100% CLI stability
$targetSessionVal = ""
$authToken = $env:BRC_TOKEN
$codeArgs = [System.Collections.Generic.List[string]]::new()

$i = 0
while ($i -lt $args.Count) {
    $item = $args[$i]
    if ($item -eq "-s" -or $item -eq "--session" -or $item -eq "-p" -or $item -eq "--port") {
        $i++
        if ($i -lt $args.Count) { $targetSessionVal = $args[$i] }
    } elseif ($item -eq "-t" -or $item -eq "--token") {
        $i++
        if ($i -lt $args.Count) { $authToken = $args[$i] }
    } else {
        $codeArgs.Add($item)
    }
    $i++
}

# Port Scanner Helper (analogous to adb devices)

function Get-BlenderSessions {
    param([int[]]$ScanPorts = (8180..8195))
    $activeList = [System.Collections.Generic.List[string]]::new()

    foreach ($pt in $ScanPorts) {
        try {
            $tcp = New-Object System.Net.Sockets.TcpClient
            $async = $tcp.BeginConnect("127.0.0.1", $pt, $null, $null)
            $wait = $async.AsyncWaitHandle.WaitOne(80, $false)
            if ($wait -and $tcp.Connected) {
                $tcp.Close()
                try {
                    $req = [System.Net.WebRequest]::Create("http://127.0.0.1:$pt/")
                    $req.Timeout = 300
                    $resp = $req.GetResponse()
                    $resp.Close()
                    [void]$activeList.Add("127.0.0.1:$pt")
                } catch [System.Net.WebException] {
                    $wresp = $_.Exception.Response
                    if ($null -ne $wresp) {
                        $code = [int]$wresp.StatusCode
                        if ($code -eq 400 -or $code -eq 401 -or $code -eq 200) {
                            [void]$activeList.Add("127.0.0.1:$pt")
                        }
                        $wresp.Close()
                    }
                }
            } else {
                $tcp.Close()
            }
        } catch {}
    }
    return ,($activeList.ToArray())
}


# Handle 'sessions' subcommand (analogous to 'adb devices')
if ($codeArgs.Count -gt 0 -and $codeArgs[0] -eq "sessions") {
    Write-Host "List of Blender sessions attached" -ForegroundColor Cyan
    $sessionsList = Get-BlenderSessions
    if ($sessionsList.Count -eq 0) {
        Write-Host "No active Blender sessions found on localhost." -ForegroundColor Yellow
    } else {
        foreach ($sess in $sessionsList) {
            Write-Host "$sess`tdevice (Blender Remote Console)" -ForegroundColor Green
        }
    }
    exit 0
}

# Print Help if no arguments
if ($codeArgs.Count -eq 0) {
    Write-Host "Blender Remote Console CLI (brc)" -ForegroundColor Cyan
    Write-Host "Usage:"
    Write-Host "  brc sessions                       List attached Blender sessions"
    Write-Host "  brc <python_code>                  Execute single-line Python code"
    Write-Host "  brc <script.py>                    Execute a Python script file"
    Write-Host "  brc -s <port|host:port> <code|file> Target a specific Blender session"
    Write-Host "  brc -t <token> <code|file>         Pass authentication token"
    Write-Host ""
    Write-Host "Examples:"
    Write-Host "  brc sessions"
    Write-Host "  brc `"print(bpy.context.scene.name)`""
    Write-Host "  brc my_script.py"
    Write-Host "  brc -s 8182 `"bpy.ops.mesh.primitive_cube_add()`""
    exit 0
}

# Target Session Resolution (Auto-detect like adb)
$targetHostPort = ""
if ($targetSessionVal) {
    if ($targetSessionVal -match "^\d+$") {
        $targetHostPort = "127.0.0.1:$targetSessionVal"
    } else {
        $targetHostPort = $targetSessionVal
    }
} else {
    $activeSessions = Get-BlenderSessions
    if ($activeSessions.Count -eq 0) {
        Write-Host "brc: error: no Blender sessions found on localhost." -ForegroundColor Red
        Write-Host "Make sure Blender is running and 'Start Server' is enabled in Remote Console N-panel." -ForegroundColor Yellow
        exit 1
    } elseif ($activeSessions.Count -eq 1) {
        $targetHostPort = $activeSessions[0]
    } else {
        Write-Host "brc: error: more than one Blender session active:" -ForegroundColor Red
        foreach ($sess in $activeSessions) {
            Write-Host "  $sess" -ForegroundColor Yellow
        }
        Write-Host "Use 'brc -s <port> ...' to target a specific session." -ForegroundColor Cyan
        exit 1
    }
}

# Combine input arguments as payload (File or Raw String)
$inputArg = $codeArgs -join " "
$codePayload = ""

if (Test-Path -Path $inputArg -PathType Leaf) {
    $codePayload = Get-Content -Path $inputArg -Raw -Encoding UTF8
} else {
    $codePayload = $inputArg
}

if (-not $codePayload.Trim()) {
    Write-Host "brc: error: no Python code or file specified." -ForegroundColor Red
    exit 1
}

# Execute HTTP Request to target Blender session
$url = "http://$targetHostPort/"
try {
    $req = [System.Net.WebRequest]::Create($url)
    $req.Method = "POST"
    $req.ContentType = "text/plain; charset=utf-8"

    if ($authToken) {
        $req.Headers.Add("X-Auth-Token", $authToken)
    }

    $utf8Bytes = [System.Text.Encoding]::UTF8.GetBytes($codePayload)
    $req.ContentLength = $utf8Bytes.Length

    $reqStream = $req.GetRequestStream()
    $reqStream.Write($utf8Bytes, 0, $utf8Bytes.Length)
    $reqStream.Close()

    $response = $req.GetResponse()
    $respStream = $response.GetResponseStream()
    $reader = New-Object System.IO.StreamReader($respStream, [System.Text.Encoding]::UTF8)
    $responseText = $reader.ReadToEnd()
    $reader.Close()
    $response.Close()

    if ($responseText) {
        [System.Console]::Write($responseText)
    }
    exit 0
} catch [System.Net.WebException] {
    $wresp = $_.Exception.Response
    if ($null -ne $wresp) {
        $respStream = $wresp.GetResponseStream()
        $reader = New-Object System.IO.StreamReader($respStream, [System.Text.Encoding]::UTF8)
        $errText = $reader.ReadToEnd()
        $reader.Close()
        $wresp.Close()

        if ($errText) {
            [System.Console]::Error.Write($errText)
        } else {
            Write-Host "brc: HTTP Error $([int]$wresp.StatusCode)" -ForegroundColor Red
        }
    } else {
        Write-Host "brc: error connecting to $targetHostPort - $($_.Exception.Message)" -ForegroundColor Red
    }
    exit 1
} catch {
    Write-Host "brc: error executing request - $($_.Exception.Message)" -ForegroundColor Red
    exit 1
}
