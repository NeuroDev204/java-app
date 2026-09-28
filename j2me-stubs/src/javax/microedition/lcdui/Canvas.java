package javax.microedition.lcdui;

public abstract class Canvas extends Displayable {
    public static final int UP = -1;
    public static final int DOWN = -2;
    public static final int LEFT = -3;
    public static final int RIGHT = -4;
    public static final int FIRE = -5;
    public static final int GAME_A = -6;
    public static final int GAME_B = -7;
    public static final int GAME_C = -8;
    public static final int GAME_D = -9;

    public static final int KEY_NUM0 = 48;
    public static final int KEY_NUM1 = 49;
    public static final int KEY_NUM2 = 50;
    public static final int KEY_NUM3 = 51;
    public static final int KEY_NUM4 = 52;
    public static final int KEY_NUM5 = 53;
    public static final int KEY_NUM6 = 54;
    public static final int KEY_NUM7 = 55;
    public static final int KEY_NUM8 = 56;
    public static final int KEY_NUM9 = 57;
    public static final int KEY_STAR = 42;
    public static final int KEY_POUND = 35;

    public Canvas() {}

    public void setFullScreenMode(boolean mode) {}
    public boolean hasPointerEvents() { return true; }
    public boolean hasPointerMotionEvents() { return true; }
    public boolean hasRepeatEvents() { return true; }

    public int getWidth() { return 240; }
    public int getHeight() { return 320; }

    public String getKeyName(int keyCode) {
        switch (keyCode) {
            case UP: return "Up";
            case DOWN: return "Down";
            case LEFT: return "Left";
            case RIGHT: return "Right";
            case FIRE: return "Select";
            case KEY_NUM0: return "0";
            case KEY_NUM1: return "1";
            case KEY_NUM2: return "2";
            case KEY_NUM3: return "3";
            case KEY_NUM4: return "4";
            case KEY_NUM5: return "5";
            case KEY_NUM6: return "6";
            case KEY_NUM7: return "7";
            case KEY_NUM8: return "8";
            case KEY_NUM9: return "9";
            case KEY_STAR: return "*";
            case KEY_POUND: return "#";
            default:
                if (keyCode >= 32 && keyCode <= 126) {
                    return String.valueOf((char)keyCode);
                }
                return "";
        }
    }

    public int getGameAction(int keyCode) {
        switch (keyCode) {
            case -1: return UP;
            case -2: return DOWN;
            case -3: return LEFT;
            case -4: return RIGHT;
            case -5: return FIRE;
            case KEY_NUM2: return UP;
            case KEY_NUM8: return DOWN;
            case KEY_NUM4: return LEFT;
            case KEY_NUM6: return RIGHT;
            case KEY_NUM5: return FIRE;
            default: return 0;
        }
    }

    public int getKeyCode(int gameAction) {
        switch (gameAction) {
            case UP: return -1;
            case DOWN: return -2;
            case LEFT: return -3;
            case RIGHT: return -4;
            case FIRE: return -5;
            default: return 0;
        }
    }

    public void repaint() {
        Display d = Display.getDisplay(null);
        if (d != null && d.getCurrent() != null && d.getCurrent() != this) {
            return;
        }
        serviceRepaints();
    }

    public void repaint(int x, int y, int width, int height) {
        Display d = Display.getDisplay(null);
        if (d != null && d.getCurrent() != null && d.getCurrent() != this) {
            return;
        }
        serviceRepaints();
    }

    public void serviceRepaints() {
        Display d = Display.getDisplay(null);
        if (d != null && d.getCurrent() != null && d.getCurrent() != this) {
            return;
        }
        paint(new Graphics());
        flushGraphics();
    }

    private native void flushGraphics();

    protected abstract void paint(Graphics g);

    protected void keyPressed(int keyCode) { System.out.println("Canvas keyPressed: " + keyCode); }
    protected void keyReleased(int keyCode) { System.out.println("Canvas keyReleased: " + keyCode); }
    protected void keyRepeated(int keyCode) { System.out.println("Canvas keyRepeated: " + keyCode); }
    protected void pointerPressed(int x, int y) { System.out.println("Canvas pointerPressed: " + x + "," + y); }
    protected void pointerReleased(int x, int y) { System.out.println("Canvas pointerReleased: " + x + "," + y); }
    protected void pointerDragged(int x, int y) { System.out.println("Canvas pointerDragged: " + x + "," + y); }
}
