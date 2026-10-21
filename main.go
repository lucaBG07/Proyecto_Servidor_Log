package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func atenderCliente(file *os.File, conn net.Conn) {
	// Cerramos el conn al terminar la funcion
	defer conn.Close()

	// Creamos un scan
	scan := bufio.NewScanner(conn)

	// scanner.Scan() devuelve true mientras haya una línea nueva para leer, y false cuando
	// el cliente se desconecta o hay un error. Por eso el for termina solo cuando el cliente se va.
	for scan.Scan() {
		linea := scan.Text()

		// Escribe en consola
		fmt.Printf("%s\n", linea)

		// Escribe en servidor.log
		fmt.Fprintf(file, "%s\n", linea)
	}

	// Verificamos que en el scan salio todo bien.
	if err := scan.Err(); err != nil {
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

	for {
		// se queda bloqueado (esperando) hasta que un cliente se conecta, y cuando eso pasa
		// devuelve una conexión (net.Conn) y un error.
		conn, err := listener.Accept()

		if err != nil {
			fmt.Printf("[Error] falla al momento de conectar: %v\n", err)
			// salta directo a la siguiente vuelta del bucle.
			continue
		}

		go atenderCliente(file, conn)
	}
}
