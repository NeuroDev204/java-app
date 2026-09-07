package javax.microedition.io;

import java.io.InputStream;
import java.io.DataInputStream;
import java.io.IOException;

public interface InputConnection extends Connection {
    public InputStream openInputStream() throws IOException;
    public DataInputStream openDataInputStream() throws IOException;
}
