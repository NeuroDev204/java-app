package javax.microedition.lcdui;

import java.io.IOException;
import java.io.InputStream;

public class Image {
    private int nativeId;

    private static final java.lang.ref.ReferenceQueue<Image> refQueue = new java.lang.ref.ReferenceQueue<>();
    private static final java.util.Set<ImageReference> refs = java.util.Collections.synchronizedSet(new java.util.HashSet<>());
    
    private static class ImageReference extends java.lang.ref.WeakReference<Image> {
        int nativeId;
        ImageReference(Image referent, java.lang.ref.ReferenceQueue<? super Image> q) {
            super(referent, q);
            nativeId = referent.nativeId;
        }
    }

    public static void pollQueue() {
        ImageReference ref;
        while ((ref = (ImageReference) refQueue.poll()) != null) {
            refs.remove(ref);
            try { destroy(ref.nativeId); } catch(Throwable t){}
        }
    }

    static {
        new Thread(new Runnable() {
            public void run() {
                while (true) {
                    try {
                        Thread.sleep(1000);
                        pollQueue();
                    } catch (Exception e) {}
                }
            }
        }).start();
    }

    public Image(int nativeId) {
        this.nativeId = nativeId;
        refs.add(new ImageReference(this, refQueue));
        pollQueue();
    }

    public int getId() {
        pollQueue();
        return nativeId;
    }

    public static Image createImage(String path) throws IOException {
        InputStream is = Image.class.getResourceAsStream(path);
        if (is == null) {
            throw new IOException("Resource not found: " + path);
        }
        return createImage(is);
    }
    public static native Image createImage(int width, int height);
    public static native Image createImage(byte[] data, int imageOffset, int imageLength);
    public static Image createImage(Image source) { return source; }
    public static native Image createImage(Image source, int x, int y, int width, int height, int transform);
    public static native Image createRGBImage(int[] rgb, int width, int height, boolean processAlpha);
    public native void getRGB(int[] rgbData, int offset, int scanlength, int x, int y, int width, int height);

    public static Image createImage(InputStream stream) throws IOException {
        if (stream == null) {
            throw new NullPointerException("InputStream is null");
        }
        byte[] buffer = new byte[4096];
        java.io.ByteArrayOutputStream baos = new java.io.ByteArrayOutputStream();
        int bytesRead;
        while ((bytesRead = stream.read(buffer)) != -1) {
            baos.write(buffer, 0, bytesRead);
        }
        byte[] data = baos.toByteArray();
        return createImage(data, 0, data.length);
    }

    public native int getWidth();
    public native int getHeight();
    public native Graphics getGraphics();
    
    protected void finalize() throws Throwable {
        try {
            destroy(nativeId);
        } finally {
            super.finalize();
        }
    }
    private static native void destroy(int id);
}
