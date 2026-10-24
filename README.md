# 📝 Servidor de logs TCP en Go

Servidor TCP simple escrito en Go que recibe líneas de texto de varios clientes al mismo tiempo y las guarda en un archivo (servidor.log), indicando de qué cliente llegó cada una.

Es un proyecto de aprendizaje: lo armé por etapas para practicar sockets, goroutines y sincronización de escrituras concurrentes.

## 📌 Qué hace
Escucha conexiones TCP en el puerto 9000.
Atiende a cada cliente en su propia goroutine, así que varios clientes pueden estar conectados a la vez.
Lee línea por línea lo que envía cada cliente (bufio.Scanner).
Registra cada línea: la muestra por consola y la agrega al final de servidor.log, con la dirección del cliente como prefijo.
Protege las escrituras con un sync.Mutex compartido, para que las líneas de distintos clientes no se mezclen.

Ejemplo de líneas en servidor.log:

[[::1]:50536] hola desde el cliente 1
[[::1]:50537] hola desde el cliente 2

El prefijo es la dirección y el puerto de origen del cliente. Con conexiones locales puede aparecer como [::1] (IPv6) o 127.0.0.1 (IPv4).

### 🧰 Tecnologías usadas
Paquete	Para qué se usa
net	Abrir el listener TCP y aceptar conexiones
bufio	Leer el flujo de cada cliente línea por línea
os	Abrir servidor.log en modo agregar al final
sync	Mutex para proteger la escritura al archivo
fmt	Escribir en consola y en el archivo

### ⚙️ Cómo ejecutarlo

Requisito: tener Go instalado (la versión mínima figura en go.mod).

Desde la carpeta del proyecto:

go run main.go

El servidor queda esperando conexiones y no muestra nada hasta que llegue un cliente. servidor.log se crea solo la primera vez. Para detenerlo: Ctrl + C.

### 🧪 Cómo probarlo

Con el servidor corriendo, abrí otra terminal y conectate como cliente.

Con Netcat
nc localhost 9000

Lo que escribas y confirmes con Enter llega al servidor.

$c = New-Object System.Net.Sockets.TcpClient("localhost", 9000)
$w = New-Object System.IO.StreamWriter($c.GetStream())
$w.AutoFlush = $true
$w.WriteLine("hola desde el cliente 1")

Cada terminal que repita estas líneas es un cliente distinto. Para cortar la conexión: $c.Close().

### 🔒 El problema de concurrencia
Qué pasaba

Cada entrada del log se escribe en dos pasos: primero el prefijo con la dirección del cliente y después el mensaje. Si dos clientes envían mucho a la vez, sus goroutines pueden intercalarse entre esos dos pasos, y el archivo queda con líneas mal formadas (dos prefijos seguidos o un mensaje sin prefijo). Es una condición de carrera.

**Cómo lo resolví**

Con un sync.Mutex compartido por todos los clientes. El candado envuelve solo las dos escrituras al archivo, así que las demás goroutines esperan únicamente ese tramo y no mientras se lee de la conexión.

**Cómo lo comprobé**

Lancé dos clientes en paralelo con 50000 líneas cada uno y busqué en el archivo las líneas que no respetaban el formato. Sin el mutex aparecieron líneas rotas; con el mutex la misma prueba no produjo ninguna.

Para repetir la prueba (con el servidor corriendo y servidor.log borrado):

```bash
1..2 | ForEach-Object {
  Start-Job -ArgumentList $_ -ScriptBlock {
    param($n)
    $c = New-Object System.Net.Sockets.TcpClient("localhost", 9000)
    $w = New-Object System.IO.StreamWriter($c.GetStream())
    $w.AutoFlush = $true
    1..50000 | ForEach-Object { $w.WriteLine("cliente $n linea $_") }
    $c.Close()
  }
} | Wait-Job | Remove-Job
```

Y para buscar las líneas que no tengan el formato esperado:

```bash
Get-Content servidor.log | Where-Object { $_ -notmatch '^\[\[::1\]:\d+\] cliente \d linea \d+$' }
```

Si no devuelve nada, todas las líneas están bien formadas.

### ⚠️ Limitaciones actuales
**Sin cierre ordenado:** al detener el servidor con Ctrl + C el programa termina de golpe.
**Sin fecha ni hora en las entradas.**
**Sin rotación: el archivo de log crece sin límite.**
**Sin niveles de log ni filtros.**
**Un único mutex serializa todas las escrituras:** alcanza para este tamaño de proyecto, pero no se pensó para mucha carga.