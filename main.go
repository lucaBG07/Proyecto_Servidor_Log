package main

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"time"
)

// Los niveles que aceptamos
var niveles = map[string]int{"DEBUG": 0, "INFO": 1, "WARN": 2, "ERROR": 3}

// El nivel minimo para el filtro
var nivelMinimo string = "WARN"

func separarNivel(texto string) (string, string) {
	// Cortar la linea en el primer espacio, en 2 partes como maximo
	partes := strings.SplitN(texto, " ", 2)

	// Si no salieron 2 partes, no hay nivel + mensaje
	if len(partes) != 2 {
		return "INFO", texto
	}

	// Pasamos el niivel a mayusculas
	nivel := strings.ToUpper(partes[0])

	_, existe := niveles[nivel]

	if existe {
		return nivel, partes[1]
	}

	// Si no esta, no era un nivel: devolvemos INFO y la linea completa
	return "INFO", texto
}

func atenderCliente(file *os.File, conn net.Conn, mu *sync.Mutex, grupo *sync.WaitGroup) {
	defer grupo.Done()
	// Cerramos el conn al terminar la funcion
	defer conn.Close()

	// Creamos un scan
	scan := bufio.NewScanner(conn)

	// scanner.Scan() devuelve true mientras haya una línea nueva para leer, y false cuando
	// el cliente se desconecta o hay un error. Por eso el for termina solo cuando el cliente se va.
	for scan.Scan() {
		linea := scan.Text()

		nivel, mensaje := separarNivel(linea)

		if niveles[nivel] < niveles[nivelMinimo] {
			continue
		}

		dire := conn.RemoteAddr()
		ahora := time.Now().Format("2006-01-02 15:04:05")

		// Escribe en consola.
		fmt.Printf("%s\n", linea)

		// Agarramos el candado: si otro cliente lo tiene, esperamos aca.
		mu.Lock()

		// Agrega la fecha y hora al principio.
		fmt.Fprint(file, ahora)

		// Prefijo [dirección del cliente]. Va dentro del candado junto con el mensaje, porque son dos
		// escrituras y no pueden separarse.
		fmt.Fprint(file, " [", dire, "] ")

		// Escribe en servidor.log
		fmt.Fprint(file, "[", nivel, "] ")
		fmt.Fprintf(file, "%s\n", mensaje)

		// Soltamos el candado para que pase el siguiente.
		mu.Unlock()
	}

	// Verificamos que en el scan salio todo bien.
	// Si la conexión se cerró a propósito al apagar el servidor, no es un error.
	err := scan.Err()
	if errors.Is(err, net.ErrClosed) {
		return
	}

	if err != nil {
		fmt.Printf("Error al leer en el puerto: %v\n", err)
	}
}

func main() {
	// Abrimos el archivo servidor.log y le pasamos:
	// os.O_WRONLY: abrirlo solo para escribir.
	// os.O_APPEND: escribir siempre al final, sin borrar lo que ya habia.
	// os.O_CREATE: crearlo si no existe.
	// Los permisos (0644): quien puede leerlo y modificarlo.
	file, err := os.OpenFile("servidor.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		fmt.Printf("[Error] al abrir el archivo: %v\n", err)
		return
	}

	// Candado compartido por todos los clientes: solo una goroutine a la vez puede escribir en el archivo.
	var mu sync.Mutex

	// Cerramos el archivo al terminar la funcion
	defer file.Close()

	// TCP funciona como un canal: el cliente escribe bytes de un lado y el servidor los lee del otro.
	// ":9000" significa “escuchar en el puerto 9000, en todas las interfaces de red de esta máquina”
	// El formato es host:puerto, y dejar el host vacío significa “todas las interfaces”.
	listener, err := net.Listen("tcp", ":9000")

	if err != nil {
		fmt.Printf("[Error]: %v\n", err)
		return
	}

	// Cerramos al terminar la funcion
	defer listener.Close()

	var conexiones []net.Conn
	var muConex sync.Mutex

	// Canal por donde el sistema operativo nos avisa de las señales.
	// Capacidad 1 para que el aviso no se pierda si todavía no estamos esperando.
	avisos := make(chan os.Signal, 1)

	// Pedimos que, cuando llegue Ctrl + C, Go mande la señal por este canal en vez de matar el programa.
	signal.Notify(avisos, os.Interrupt)

	// Goroutine aparte que espera la señal, porque main está ocupado aceptando clientes.
	go func() {
		<-avisos // Se queda bloqueada acá hasta que llegue la señal.
		fmt.Println("Señal recibida correctamente")
		listener.Close() // Cerrar el listener hace que el Accept() bloqueado devuelva net.ErrClosed y main pueda salir del bucle.

		muConex.Lock()
		for _, c := range conexiones {
			c.Close()
		}
		muConex.Unlock()
	}()

	var grupo sync.WaitGroup

	for {
		// se queda bloqueado (esperando) hasta que un cliente se conecta, y cuando eso pasa
		// devuelve una conexión (net.Conn) y un error.
		conn, err := listener.Accept()

		if errors.Is(err, net.ErrClosed) {
			break
		}

		if err != nil {
			fmt.Printf("[Error] falla al momento de conectar: %v\n", err)
			// salta directo a la siguiente vuelta del bucle.
			continue
		}

		grupo.Add(1)
		go atenderCliente(file, conn, &mu, &grupo)

		muConex.Lock()
		conexiones = append(conexiones, conn)
		muConex.Unlock()
	}

	grupo.Wait()

	fmt.Println("Servidor cerrado")
}
