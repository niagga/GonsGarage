$body = @{
  email = 'admin.demo@gonsgarage.local'
  password = 'AdminDemo123'
} | ConvertTo-Json
try {
  $r = Invoke-RestMethod -Method Post -Uri 'http://localhost:8080/api/v1/auth/login' -ContentType 'application/json' -Body $body
  $r | ConvertTo-Json -Depth 5
} catch {
  if ($_.Exception.Response) {
    [int]$_.Exception.Response.StatusCode
  } else {
    $_.Exception.Message
  }
}
