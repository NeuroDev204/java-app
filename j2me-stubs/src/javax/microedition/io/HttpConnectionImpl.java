package javax.microedition.io;

import java.io.DataInputStream;
import java.io.DataOutputStream;
import java.io.IOException;
import java.io.InputStream;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;

public class HttpConnectionImpl implements HttpConnection {
    private HttpURLConnection conn;

    public HttpConnectionImpl(String url) throws IOException {
        this.conn = (HttpURLConnection) new URL(url).openConnection();
    }

    public InputStream openInputStream() throws IOException {
        return conn.getInputStream();
    }

    public DataInputStream openDataInputStream() throws IOException {
        return new DataInputStream(conn.getInputStream());
    }

    public OutputStream openOutputStream() throws IOException {
        return conn.getOutputStream();
    }

    public DataOutputStream openDataOutputStream() throws IOException {
        return new DataOutputStream(conn.getOutputStream());
    }

    public void close() throws IOException {
        conn.disconnect();
    }

    public long getLength() {
        return conn.getContentLength();
    }

    public String getType() {
        return conn.getContentType();
    }

    public int getResponseCode() throws IOException {
        return conn.getResponseCode();
    }

    public void setRequestMethod(String method) throws IOException {
        conn.setRequestMethod(method);
    }

    public String getURL() { return conn.getURL().toString(); }
    public String getProtocol() { return conn.getURL().getProtocol(); }
    public String getHost() { return conn.getURL().getHost(); }
    public int getPort() { return conn.getURL().getPort(); }
    public String getQuery() { return conn.getURL().getQuery(); }
    public String getRequestMethod() { return conn.getRequestMethod(); }
    public String getRequestProperty(String key) { return conn.getRequestProperty(key); }
    public void setRequestProperty(String key, String value) throws IOException { conn.setRequestProperty(key, value); }
    public String getResponseMessage() throws IOException { return conn.getResponseMessage(); }
    public String getEncoding() { return conn.getContentEncoding(); }
}
