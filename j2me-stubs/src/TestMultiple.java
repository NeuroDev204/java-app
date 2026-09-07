import javax.microedition.lcdui.Graphics;

public class TestMultiple {
    public static void main(String[] args) {
        Graphics g = new Graphics();
        for (int i = 0; i < 100; i++) {
            g.setColor(i * 10);
            g.fillRect(i, i * 2, i + 10, i + 20);
        }
    }
}
