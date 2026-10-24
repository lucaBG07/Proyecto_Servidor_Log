$c = New-Object System.Net.Sockets.TcpClient("localhost", 9000)
$w = New-Object System.IO.StreamWriter($c.GetStream())
$w.AutoFlush = $true
$w.WriteLine("hola desde el cliente 1")