import javax.microedition.lcdui.Graphics;

public class MultiFillRectTest {
    public static void main(String[] args) {
        Graphics g = new Graphics();
        for (int i = 0; i < 100; i++) {
            g.fillRect(i, i * 2, 10 + i, 20 + i);
        }
    }
}
