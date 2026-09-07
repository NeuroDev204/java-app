package javax.microedition.midlet;

public abstract class MIDlet {
    protected MIDlet() {}

    protected abstract void startApp() throws MIDletStateChangeException;
    protected abstract void pauseApp();
    protected abstract void destroyApp(boolean unconditional) throws MIDletStateChangeException;

    public void notifyDestroyed() {}
    public void notifyPaused() {}
    public boolean platformRequest(String URL) throws javax.microedition.io.ConnectionNotFoundException {
        return false;
    }
    public String getAppProperty(String key) {
        return null;
    }
}
