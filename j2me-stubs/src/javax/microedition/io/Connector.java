package javax.microedition.io;

import java.io.InputStream;
import java.io.OutputStream;
import java.io.DataInputStream;
import java.io.DataOutputStream;
import java.io.IOException;

public class Connector {
    public static final int READ = 1;
    public static final int WRITE = 2;
    public static final int READ_WRITE = 3;

    public static Connection open(String name) throws IOException {
        return open(name, READ_WRITE, false);
    }

    public static Connection open(String name, int mode) throws IOException {
        return open(name, mode, false);
    }

    public static Connection open(String name, int mode, boolean timeouts) throws IOException {
        if (name.startsWith("socket://")) {
            String hostPort = name.substring(9);
            int colonIndex = hostPort.indexOf(':');
            if (colonIndex == -1) throw new IllegalArgumentException("Invalid socket url: " + name);
            String host = hostPort.substring(0, colonIndex);
            int port = Integer.parseInt(hostPort.substring(colonIndex + 1));
            return new SocketConnectionImpl(host, port);
        } else if (name.startsWith("http://") || name.startsWith("https://")) {
            return new HttpConnectionImpl(name);
        }
        throw new ConnectionNotFoundException("Protocol not supported: " + name);
    }

    public static DataInputStream openDataInputStream(String name) throws IOException {
        Connection c = open(name, READ);
        if (c instanceof InputConnection) {
            return ((InputConnection) c).openDataInputStream();
        }
        throw new IOException("Not an InputConnection: " + name);
    }

    public static DataOutputStream openDataOutputStream(String name) throws IOException {
        Connection c = open(name, WRITE);
        if (c instanceof OutputConnection) {
            return ((OutputConnection) c).openDataOutputStream();
        }
        throw new IOException("Not an OutputConnection: " + name);
    }

    public static InputStream openInputStream(String name) throws IOException {
        return openDataInputStream(name);
    }

    public static OutputStream openOutputStream(String name) throws IOException {
        return openDataOutputStream(name);
    }
}
