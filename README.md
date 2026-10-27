# 📝 Servidor de logs TCP en Go

Servidor TCP simple escrito en Go que recibe líneas de texto de varios clientes al mismo tiempo y las guarda en un archivo (servidor.log), indicando de qué cliente llegó cada una.

Es un proyecto de aprendizaje: lo armé por etapas para practicar sockets, goroutines y sincronización de escrituras concurrentes.

## 📌 Qué hace
- Escucha conexiones TCP en el puerto 9000.
- Atiende a cada cliente en su propia goroutine, así que varios clientes pueden estar conectados a la vez.
- Lee línea por línea lo que envía cada cliente (bufio.Scanner).
- Registra cada línea: la muestra por consola y la agrega al final de servidor.log, con fecha, hora y dirección del cliente como prefijo.
- Protege las escrituras con un sync.Mutex compartido, para que las líneas de distintos clientes no se mezclen.
- Se apaga de forma ordenada al recibir Ctrl + C: deja de aceptar clientes, cierra las conexiones abiertas, espera a que cada goroutine termine y recién entonces cierra el archivo.

Ejemplo de líneas en servidor.log:

2026-10-27 15:15:53 [[::1]:50536] hola desde el cliente 1
2026-10-27 15:15:54 [[::1]:50537] hola desde el cliente 2

### 🧰 Tecnologías usadas
Paquete	  |   Para qué se usa
          |   
net	      |   Abrir el listener TCP y aceptar conexiones
bufio	  |   Leer el flujo de cada cliente línea por línea
os	      |   Abrir servidor.log en modo agregar al final
os/signal |	  Detectar Ctrl + C para iniciar el cierre ordenado
sync	  |   Mutex para proteger recursos compartidos y WaitGroup para esperar a los clientes
time	  |   Fecha y hora de cada entrada
errors	  |   Distinguir un cierre intencional (net.ErrClosed) de un error real
fmt	      |   Escribir en consola y en el archivo

### ⚙️ Cómo ejecutarlo

Requisito: tener Go instalado (la versión mínima figura en go.mod).

Desde la carpeta del proyecto:

go run main.go

El servidor queda esperando conexiones y no muestra nada hasta que llegue un cliente. servidor.log se crea solo la primera vez. Para detenerlo, Ctrl + C.

### 🧪 Cómo probarlo

Con el servidor corriendo, abrí otra terminal y conectate como cliente.

**Con Netcat**
nc localhost 9000

Lo que escribas y confirmes con Enter llega al servidor.

**Con PowerShell (Windows, sin instalar nada)**
$c = New-Object System.Net.Sockets.TcpClient("localhost", 9000)
$w = New-Object System.IO.StreamWriter($c.GetStream())
$w.AutoFlush = $true
$w.WriteLine("hola desde el cliente 1")

Cada terminal que repita estas líneas es un cliente distinto. Para cortar la conexión: $c.Close().

### 🔒 El problema de concurrencia
**Qué pasaba**

Cada entrada del log se escribe en varios pasos: la hora, el prefijo con la dirección del cliente y el mensaje. Si dos clientes envían mucho a la vez, sus goroutines pueden intercalarse entre esos pasos, y el archivo queda con líneas mal formadas (prefijos seguidos o mensajes sin prefijo). Es una condición de carrera.

**Cómo lo resolví**

Con un sync.Mutex compartido por todos los clientes. El candado envuelve solo las escrituras al archivo, así que las demás goroutines esperan únicamente ese tramo y no mientras se lee de la conexión.

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
Get-Content servidor.log | Where-Object { $_ -notmatch '^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2} \[\[::1\]:\d+\] cliente \d linea \d+$' }
```

Si no devuelve nada, todas las líneas están bien formadas.

### 🛑 Cierre ordenado

Al apretar Ctrl + C, el programa no se corta de golpe. En cambio:

1- Detecta la señal: una goroutine aparte espera la señal de interrupción (os/signal) a través de un canal.
2- Deja de aceptar clientes: cierra el listener, lo que hace que el Accept() bloqueado devuelva net.ErrClosed y el bucle principal termine.
3- Cierra las conexiones abiertas: el servidor guarda las conexiones en una lista (protegida con su propio mutex) y las cierra todas. Así el Scan() de cada cliente deja de esperar y su goroutine termina.
4- Espera a los clientes: un sync.WaitGroup cuenta las goroutines activas, y main espera a que todas terminen.
5- Cierra el archivo al final, cuando ya nadie está escribiendo.

Un error de lectura causado por este cierre intencional no se muestra como falla: se distingue con errors.Is(err, net.ErrClosed).

Lo comprobé apretando Ctrl + C en medio de dos clientes enviando 50000 líneas cada uno: el servidor se cerró solo y, al revisar servidor.log con el comando de búsqueda anterior, ninguna línea quedó cortada.

### ⚠️ Limitaciones actuales
**Cierre con un caso límite:** un cliente que se conecte justo en el instante del Ctrl + C podría quedar fuera de la lista de conexiones y no cerrarse, dejando al servidor esperándolo.
**La lista de conexiones crece mientras el servidor esté vivo:** las conexiones cerradas no se quitan de ella.
**Sin rotación:** el archivo de log crece sin límite.
**Sin niveles de log ni filtros.**
**Un único mutex serializa todas las escrituras al archivo:** alcanza para este tamaño de proyecto, pero no se pensó para mucha carga.