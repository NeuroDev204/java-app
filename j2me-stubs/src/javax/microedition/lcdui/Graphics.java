package javax.microedition.lcdui;

public class Graphics {
    public int nativeTargetId = 0; // 0 = main display, >0 = image ID

    public Graphics() {}
    public Graphics(int targetId) { this.nativeTargetId = targetId; }
    public static final int HCENTER = 1;
    public static final int VCENTER = 2;
    public static final int LEFT = 4;
    public static final int RIGHT = 8;
    public static final int TOP = 16;
    public static final int BOTTOM = 32;
    public static final int BASELINE = 64;

    public static final int SOLID = 0;
    public static final int DOTTED = 1;

    public native void fillRect(int x, int y, int width, int height);
    public native void drawRect(int x, int y, int width, int height);
    public native void drawLine(int x1, int y1, int x2, int y2);
    public native void setColor(int rgb);
    public native void setColor(int red, int green, int blue);
    public native void drawString(String str, int x, int y, int anchor);
    public native void drawSubstring(String str, int offset, int len, int x, int y, int anchor);
    public native void drawChar(char ch, int x, int y, int anchor);
    public native void drawChars(char[] data, int offset, int length, int x, int y, int anchor);

    public native void drawImage(Image img, int x, int y, int anchor);
    public native void drawRegion(Image src, int x_src, int y_src, int width, int height, int transform, int x_dest, int y_dest, int anchor);
    public native void drawRGB(int[] rgbData, int offset, int scanlength, int x, int y, int width, int height, boolean processAlpha);
    
    public void copyArea(int x_src, int y_src, int width, int height, int x_dest, int y_dest, int anchor) {}
    public void displayHints(int hints) {}

    public native void setClip(int x, int y, int width, int height);
    public native void clipRect(int x, int y, int width, int height);
    public native int getClipX();
    public native int getClipY();
    public native int getClipWidth();
    public native int getClipHeight();

    public native void translate(int x, int y);
    public native int getTranslateX();
    public native int getTranslateY();

    public native void fillArc(int x, int y, int width, int height, int startAngle, int arcAngle);
    public native void drawArc(int x, int y, int width, int height, int startAngle, int arcAngle);
    public native void fillRoundRect(int x, int y, int width, int height, int arcWidth, int arcHeight);
    public native void drawRoundRect(int x, int y, int width, int height, int arcWidth, int arcHeight);
    public native void fillTriangle(int x1, int y1, int x2, int y2, int x3, int y3);

    public native Font getFont();
    public native void setFont(Font font);
}
