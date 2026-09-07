package javax.microedition.io;

import java.io.IOException;

public interface ContentConnection extends StreamConnection {
    public String getType();
    public String getEncoding();
    public long getLength();
}
