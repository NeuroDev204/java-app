package CLib;

import java.io.ByteArrayOutputStream;
import java.io.DataInputStream;
import java.io.InputStream;
import javax.microedition.lcdui.Image;

public class mImage {
    public Image image;

    public mImage() {}

    private static int getScale() {
        try {
            Class<?> gfxClass = Class.forName("CLib.mGraphics");
            java.lang.reflect.Field field = gfxClass.getField("zoomLevel");
            int val = field.getInt(null);
            return val > 0 ? val : 1;
        } catch (Throwable t) {
            try {
                Class<?> bxClass = Class.forName("bx");
                java.lang.reflect.Field field = bxClass.getField("b");
                int val = field.getInt(null);
                return val > 0 ? val : 1;
            } catch (Throwable t2) {
                return 1;
            }
        }
    }

    public static InputStream getResource(String path) {
        if (path == null) return null;
        String cleanPath = path.startsWith("/") ? path.substring(1) : path;
        String slashPath = "/" + cleanPath;
        InputStream is = null;
        if (com.emulator.Main.j2meClassLoader != null) {
            is = com.emulator.Main.j2meClassLoader.getResourceAsStream(cleanPath);
            if (is == null) {
                is = com.emulator.Main.j2meClassLoader.getResourceAsStream(slashPath);
            }
        }
        if (is == null) {
            ClassLoader ccl = Thread.currentThread().getContextClassLoader();
            if (ccl != null) {
                is = ccl.getResourceAsStream(cleanPath);
                if (is == null) {
                    is = ccl.getResourceAsStream(slashPath);
                }
            }
        }
        if (is == null) {
            is = com.emulator.Main.class.getResourceAsStream(slashPath);
        }
        if (is == null) {
            is = mImage.class.getResourceAsStream(slashPath);
        }
        return is;
    }

    public static String getLink(String path) {
        return path;
    }

    public void clearData() {}

    public static DataInputStream openFile(String path) {
        if (path == null) return null;
        String cleanPath = path.startsWith("/") ? path.substring(1) : path;
        InputStream is = getResource(cleanPath);
        if (is == null) {
            is = getResource("x" + getScale() + "/" + cleanPath);
        }
        if (is == null && path.endsWith(".img")) {
            String p2 = path.substring(0, path.length() - 4) + ".png";
            is = getResource(p2);
            if (is == null) {
                is = getResource("x" + getScale() + "/" + (p2.startsWith("/") ? p2.substring(1) : p2));
            }
        }
        if (is == null && path.endsWith("img")) {
            String p2 = path.substring(0, path.length() - 3) + "png";
            is = getResource(p2);
            if (is == null) {
                is = getResource("x" + getScale() + "/" + (p2.startsWith("/") ? p2.substring(1) : p2));
            }
        }
        if (is != null) {
            return new DataInputStream(is);
        }
        return null;
    }

    public static String replaceIMGtoPNG(String str, String from, String to) {
        if (str == null) return null;
        if (str.endsWith(from)) {
            return str.substring(0, str.length() - from.length()) + to;
        }
        return str;
    }

    public static mImage createEncodedImage(String path) {
        mImage instance = new mImage();
        try {
            String fullPath = "/x" + getScale() + (path.startsWith("/") ? path : "/" + path);
            DataInputStream dis = openFile(fullPath);
            if (dis == null) {
                dis = openFile(path);
            }
            if (dis != null) {
                ByteArrayOutputStream baos = new ByteArrayOutputStream();
                byte[] buf = new byte[4096];
                int n;
                while ((n = dis.read(buf)) != -1) {
                    baos.write(buf, 0, n);
                }
                dis.close();
                byte[] data = baos.toByteArray();
                for (int i = 0; i < data.length; i++) {
                    data[i] = (byte)(data[i] ^ 3);
                }
                instance.image = Image.createImage(data, 0, data.length);
            }
        } catch (Throwable t) {}
        return instance.image != null ? instance : null;
    }

    public static mImage createImage(String path) {
        if (path == null) return null;
        if (path.endsWith("img")) {
            String pngPath = path.substring(0, path.length() - 3) + "png";
            mImage res = createImage(pngPath);
            if (res != null) return res;
            return createEncodedImage(path);
        }
        try {
            mImage instance = new mImage();
            String fullPath = "/x" + getScale() + (path.startsWith("/") ? path : "/" + path);
            DataInputStream dis = openFile(fullPath);
            if (dis == null) {
                dis = openFile(path);
            }
            if (dis != null) {
                ByteArrayOutputStream baos = new ByteArrayOutputStream();
                byte[] buf = new byte[4096];
                int n;
                while ((n = dis.read(buf)) != -1) {
                    baos.write(buf, 0, n);
                }
                dis.close();
                byte[] data = baos.toByteArray();
                instance.image = Image.createImage(data, 0, data.length);
            } else {
                instance.image = Image.createImage(fullPath);
            }
            return instance.image != null ? instance : null;
        } catch (Exception e) {
            return null;
        }
    }

    public static mImage createImageAll(String path) {
        return createImage(path);
    }

    public static mImage createImage(int width, int height) {
        mImage instance = new mImage();
        int scale = getScale();
        instance.image = Image.createImage(width * scale, height * scale);
        return instance;
    }

    public static mImage createImage(byte[] data, int off, int len, String name) {
        mImage instance = new mImage();
        if (data != null && data.length > 0) {
            try {
                instance.image = Image.createImage(data, off, len);
            } catch (Exception e) {}
        }
        return instance;
    }

    public static mImage createImage(byte[] data, int off, int len) {
        return createImage(data, off, len, "");
    }

    public static mImage createImage(byte[] data) {
        if (data == null) return null;
        return createImage(data, 0, data.length, "");
    }

    public TemGraphics getGraphics() {
        TemGraphics tg = new TemGraphics();
        if (this.image != null) {
            tg.g = this.image.getGraphics();
        }
        return tg;
    }

    public static int getImageWidth(Image img) {
        int scale = getScale();
        return img != null ? img.getWidth() / scale : 0;
    }

    public int getWith() {
        return getImageWidth(this.image);
    }

    public int getWidth() {
        return getImageWidth(this.image);
    }

    public int getHeight() {
        return getImageHeight(this.image);
    }

    public static int getImageHeight(Image img) {
        int scale = getScale();
        return img != null ? img.getHeight() / scale : 0;
    }

    public void getRGB(int[] rgbData, int offset, int scanlength, int x, int y, int width, int height) {
        if (this.image != null) {
            this.image.getRGB(rgbData, offset, scanlength, x, y, width, height);
        }
    }

    public static mImage createRGBImage(int[] rgb, int width, int height, boolean processAlpha) {
        mImage instance = new mImage();
        instance.image = Image.createRGBImage(rgb, width, height, processAlpha);
        return instance;
    }
}
