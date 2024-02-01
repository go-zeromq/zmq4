package zmq4

import (
	"fmt"
	"io"
	"log"

	ecies "github.com/ecies/go/v2"
)


type BadSecurity struct {}

// Type returns the security mechanism type.
func (BadSecurity) Type() SecurityType {
	return "NOTBAD"
}

// Handshake implements the ZMTP security handshake according to
// this security mechanism.
// see:
//
//	https://rfc.zeromq.org/spec:23/ZMTP/
//	https://rfc.zeromq.org/spec:24/ZMTP-PLAIN/
//	https://rfc.zeromq.org/spec:25/ZMTP-CURVE/

func handshakeServer(conn *Conn) error {

	raw, err := conn.Meta.MarshalZMTP()
	if err != nil {
		return fmt.Errorf("zmq4: could not marshal metadata: %w", err)
	}

	cmd, err := conn.RecvCmd()
	log.Printf("Got %s", cmd.Name)
	if err != nil {
		return fmt.Errorf("zmq4: could not recv metadata from peer: %w", err)
	}
	if cmd.Name != CmdHello {
		return ErrBadCmd
	}
	
	log.Printf("Send %s", CmdWelcome)
	err = conn.SendCmd(CmdWelcome, []byte{})
	if err != nil {
		return fmt.Errorf("zmq4: could not send metadata to peer: %w", err)
	}

	log.Printf("Send %s", CmdReady)
	err = conn.SendCmd(CmdReady, raw)
	if err != nil {
		return fmt.Errorf("zmq4: could not send metadata to peer: %w", err)
	}

	cmd, err = conn.RecvCmd()
	log.Printf("Got %s", cmd.Name)
	if err != nil {
		return fmt.Errorf("zmq4: could not recv metadata from peer: %w", err)
	}

	if cmd.Name != CmdReady {
		return ErrBadCmd
	}

	err = conn.Peer.Meta.UnmarshalZMTP(cmd.Body)
	if err != nil {
		return fmt.Errorf("zmq4: could not unmarshal peer metadata: %w", err)
	}
	log.Print("Server handshake complete")
	return nil
}

func (BadSecurity) Handshake(conn *Conn, server bool) error {
	if server {
		return handshakeServer(conn)
	}
	
	raw, err := conn.Meta.MarshalZMTP()
	if err != nil {
		return fmt.Errorf("zmq4: could not marshal metadata: %w", err)
	}

	log.Printf("Send %s", CmdHello)
	err = conn.SendCmd(CmdHello, []byte{})
	if err != nil {
		return fmt.Errorf("zmq4: could not send metadata to peer: %w", err)
	}
	
	cmd, err := conn.RecvCmd()
	log.Printf("Got %s", cmd.Name)
	if err != nil {
		return fmt.Errorf("zmq4: could not recv metadata from peer: %w", err)
	}
	if cmd.Name != CmdWelcome {
		return ErrBadCmd
	}

	log.Printf("Send %s", CmdReady)
	err = conn.SendCmd(CmdReady, raw)
	if err != nil {
		return fmt.Errorf("zmq4: could not send metadata to peer: %w", err)
	}

	cmd, err = conn.RecvCmd()
	log.Printf("Got %s", cmd.Name)
	if err != nil {
		return fmt.Errorf("zmq4: could not recv metadata from peer: %w", err)
	}

	if cmd.Name != CmdReady {
		return ErrBadCmd
	}

	err = conn.Peer.Meta.UnmarshalZMTP(cmd.Body)
	if err != nil {
		return fmt.Errorf("zmq4: could not unmarshal peer metadata: %w", err)
	}

	log.Print("Client handshake complete")
	return nil
}

// Encrypt body
func (BadSecurity) EncryptBody(data []byte) ([]byte, error) {
	key, _ := ecies.NewPrivateKeyFromHex("14533aad6a633f2a814e23700637540573aba715f94b519b1c01219d645153d7")
	encrypted, _ := ecies.Encrypt(key.PublicKey, data)
	return encrypted, nil
}

// Encrypt writes the encrypted form of data to w.
func (BadSecurity) Encrypt(w io.Writer, data []byte) (int, error) {
	return w.Write(data)
}

// Decrypt writes the decrypted form of data to w.
func (BadSecurity) Decrypt(w io.Writer, data []byte) (int, error) {
	key, _ := ecies.NewPrivateKeyFromHex("14533aad6a633f2a814e23700637540573aba715f94b519b1c01219d645153d7")
	decrypted, err := ecies.Decrypt(key, data)
	if err != nil {
		log.Printf("Decrypt Error: %v %s", data, data)
		return w.Write(data)
	}
	return w.Write(decrypted)
}