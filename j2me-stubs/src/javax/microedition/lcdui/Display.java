package javax.microedition.lcdui;

import javax.microedition.midlet.MIDlet;

public class Display {
    private static Display instance = new Display();
    private Displayable current;

    public static Display getDisplay(MIDlet m) {
        return instance;
    }

    public Displayable getCurrent() {
        return current;
    }

    public void setCurrent(Displayable nextDisplayable) {
        this.current = nextDisplayable;
        nSetCurrent(nextDisplayable);
    }

    private static native void nSetCurrent(Displayable d);

    public boolean callSerially(Runnable r) {
        if (r != null) {
            r.run();
            return true;
        }
        return false;
    }

    public native void vibrate(int duration);
    public native boolean flashBacklight(int duration);
}
