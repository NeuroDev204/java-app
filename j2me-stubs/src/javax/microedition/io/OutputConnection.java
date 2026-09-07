package javax.microedition.io;

import java.io.OutputStream;
import java.io.DataOutputStream;
import java.io.IOException;

public interface OutputConnection extends Connection {
    public OutputStream openOutputStream() throws IOException;
    public DataOutputStream openDataOutputStream() throws IOException;
}
