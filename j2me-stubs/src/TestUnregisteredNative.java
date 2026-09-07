import javax.microedition.lcdui.Graphics;

public class TestUnregisteredNative {
    public static void main(String[] args) {
        Graphics g = new Graphics();
        try {
            g.drawRect(0, 0, 10, 10);
        } catch (UnsatisfiedLinkError e) {
            System.out.println("CAUGHT_UNSATISFIED_LINK_ERROR");
        }
    }
}
