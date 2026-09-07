package javax.microedition.io;

import java.io.IOException;

public interface HttpConnection extends ContentConnection {
    public static final String GET = "GET";
    public static final String POST = "POST";
    public static final String HEAD = "HEAD";

    public static final int HTTP_OK = 200;
    public static final int HTTP_UNAUTHORIZED = 401;
    public static final int HTTP_NOT_FOUND = 404;

    public String getURL();
    public String getProtocol();
    public String getHost();
    public int getPort();
    public String getQuery();
    public String getRequestMethod();
    public void setRequestMethod(String method) throws IOException;
    public String getRequestProperty(String key);
    public void setRequestProperty(String key, String value) throws IOException;
    public int getResponseCode() throws IOException;
    public String getResponseMessage() throws IOException;
}
