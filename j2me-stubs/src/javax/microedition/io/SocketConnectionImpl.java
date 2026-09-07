package javax.microedition.io;

import java.io.DataInputStream;
import java.io.DataOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.net.Socket;

public class SocketConnectionImpl implements SocketConnection {
    private Socket socket;

    public SocketConnectionImpl(String host, int port) throws IOException {
        this.socket = new Socket(host, port);
        try {
            this.socket.setTcpNoDelay(true); // Tối đa hóa tốc độ, giảm ping (Disable Nagle's algorithm)
            this.socket.setKeepAlive(true);
            this.socket.setReceiveBufferSize(65536);
            this.socket.setSendBufferSize(65536);
        } catch (Exception e) {}
    }

    public InputStream openInputStream() throws IOException {
        return socket.getInputStream();
    }

    public DataInputStream openDataInputStream() throws IOException {
        return new DataInputStream(socket.getInputStream());
    }

    public OutputStream openOutputStream() throws IOException {
        return socket.getOutputStream();
    }

    public DataOutputStream openDataOutputStream() throws IOException {
        return new DataOutputStream(socket.getOutputStream());
    }

    public void close() throws IOException {
        socket.close();
    }

    public String getAddress() throws IOException {
        return socket.getInetAddress().getHostAddress();
    }
    public String getLocalAddress() throws IOException {
        return socket.getLocalAddress().getHostAddress();
    }
    public int getLocalPort() throws IOException {
        return socket.getLocalPort();
    }
    public int getPort() throws IOException {
        return socket.getPort();
    }
    public int getSocketOption(byte option) throws IllegalArgumentException, IOException {
        return 0;
    }
    public void setSocketOption(byte option, int value) throws IllegalArgumentException, IOException {
    }
}
