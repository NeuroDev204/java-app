package com.emulator;

import java.io.File;
import java.net.URL;
import java.net.URLClassLoader;
import javax.microedition.midlet.MIDlet;

public class Main {
    private static MIDlet currentMidlet;

    public static void startApp(String jarPath) {
        try {
            System.out.println("[Main] Loading J2ME JAR: " + jarPath);
            File file = new File(jarPath);
            if (!file.exists()) {
                System.err.println("[Main] ERROR: JAR file does not exist: " + jarPath);
                return;
            }

            URLClassLoader classLoader = new URLClassLoader(
                new URL[]{ file.toURI().toURL() },
                Main.class.getClassLoader()
            );

            Thread.currentThread().setContextClassLoader(classLoader);

            Class<?> midletClass = classLoader.loadClass("com.silverknight.TemMidlet");
            System.out.println("[Main] Loaded MIDlet class: " + midletClass.getName());

            MIDlet midlet = (MIDlet) midletClass.getDeclaredConstructor().newInstance();
            currentMidlet = midlet;

            java.lang.reflect.Method startAppMethod = midletClass.getDeclaredMethod("startApp");
            startAppMethod.setAccessible(true);
            System.out.println("[Main] Invoking startApp()...");
            startAppMethod.invoke(midlet);
            System.out.println("[Main] MIDlet startApp() executed successfully.");

            // Background Garbage Collector to keep native VRAM memory low
            new Thread(new Runnable() {
                public void run() {
                    while (true) {
                        try {
                            Thread.sleep(5000); // 5 seconds
                            System.gc(); // Force Java to collect unused Image objects
                        } catch (InterruptedException e) {}
                    }
                }
            }).start();
        } catch (Exception e) {
            System.err.println("[Main] Exception starting MIDlet:");
            e.printStackTrace();
        }
    }

    public static void main(String[] args) {
        String path = (args != null && args.length > 0) ? args[0] : "/home/neuro/Downloads/KnightOnline_402.jar";
        startApp(path);
    }
}
