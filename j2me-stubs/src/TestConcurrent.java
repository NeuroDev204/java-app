import javax.microedition.lcdui.Graphics;

public class TestConcurrent {
    public static void main(String[] args) {
        final Graphics g = new Graphics();
        Thread[] threads = new Thread[5];
        for (int i = 0; i < 5; i++) {
            final int id = i;
            threads[i] = new Thread(new Runnable() {
                public void run() {
                    for (int j = 0; j < 20; j++) {
                        g.setColor(id * 1000 + j);
                        g.fillRect(id * 10, j * 5, 50, 50);
                    }
                }
            });
            threads[i].start();
        }
        for (int i = 0; i < 5; i++) {
            try {
                threads[i].join();
            } catch (InterruptedException e) {
                e.printStackTrace();
            }
        }
    }
}
