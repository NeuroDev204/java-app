import java.io.ByteArrayOutputStream;
import java.io.DataInputStream;
import java.io.InputStream;
import javax.microedition.lcdui.Image;

public final class aq {
    public Image a;

    public aq() {}

    private static int getScale() {
        try {
            Class<?> bxClass = Class.forName("bx");
            java.lang.reflect.Field field = bxClass.getField("b");
            int val = field.getInt(null);
            return val > 0 ? val : 1;
        } catch (Throwable t) {
            return 1;
        }
    }

    public static DataInputStream b(String path) {
        if (path == null) return null;
        if (!path.startsWith("/")) {
            path = "/" + path;
        }
        InputStream is = com.emulator.Main.class.getResourceAsStream(path);
        if (is == null && path.endsWith(".img")) {
            String p2 = path.substring(0, path.length() - 4) + ".png";
            is = com.emulator.Main.class.getResourceAsStream(p2);
        }
        if (is == null && path.endsWith("img")) {
            String p2 = path.substring(0, path.length() - 3) + "png";
            is = com.emulator.Main.class.getResourceAsStream(p2);
        }
        if (is == null) {
            is = aq.class.getResourceAsStream(path);
        }
        if (is != null) {
            return new DataInputStream(is);
        }
        return null;
    }

    private static aq c(String path) {
        aq instance = new aq();
        try {
            String fullPath = "/x" + getScale() + path;
            DataInputStream dis = b(fullPath);
            if (dis == null) {
                dis = b(path);
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
                instance.a = Image.createImage(data, 0, data.length);
            }
        } catch (Throwable t) {}
        return instance.a != null ? instance : null;
    }

    public static aq a(String path) {
        if (path == null) return null;
        if (path.endsWith("img")) {
            String pngPath = path.substring(0, path.length() - 3) + "png";
            aq res = c(pngPath);
            if (res != null) return res;
            return c(path);
        }
        try {
            aq instance = new aq();
            String fullPath = "/x" + getScale() + path;
            DataInputStream dis = b(fullPath);
            if (dis == null) {
                dis = b(path);
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
                instance.a = Image.createImage(data, 0, data.length);
            } else {
                instance.a = Image.createImage(fullPath);
            }
            return instance.a != null ? instance : null;
        } catch (Exception e) {
            return null;
        }
    }

    public static aq a(byte[] data) {
        aq instance = new aq();
        if (data != null && data.length > 0) {
            try {
                instance.a = Image.createImage(data, 0, data.length);
            } catch (Exception e) {}
        }
        return instance;
    }

    public static int a(Image img) {
        int scale = getScale();
        return img != null ? img.getWidth() / scale : 0;
    }

    public static int b(Image img) {
        int scale = getScale();
        return img != null ? img.getHeight() / scale : 0;
    }
}
