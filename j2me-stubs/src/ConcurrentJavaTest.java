import javax.microedition.lcdui.Graphics;

public class ConcurrentJavaTest {
    public static void main(String[] args) {
        final Graphics g = new Graphics();
        Thread[] threads = new Thread[10];
        for (int t = 0; t < 10; t++) {
            final int threadId = t;
            threads[t] = new Thread(new Runnable() {
                public void run() {
                    for (int i = 0; i < 50; i++) {
                        g.fillRect(threadId, i, 5, 5);
                    }
                }
            });
            threads[t].start();
        }
        for (int t = 0; t < 10; t++) {
            try {
                threads[t].join();
            } catch (InterruptedException e) {
                e.printStackTrace();
            }
        }
    }
}
