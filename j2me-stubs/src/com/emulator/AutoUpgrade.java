package com.emulator;

import java.lang.reflect.Field;
import java.lang.reflect.Method;
import java.util.ArrayList;
import java.util.List;

/**
 * AutoUpgrade: Java Runtime Hook bridging the active running J2ME JVM game session to Go.
 * Provides reflection-based inventory inspection, lucky clover / material counting,
 * and background automated equipment upgrading via packet dispatching.
 */
public class AutoUpgrade {

    // Upgrade routine state variables
    private static volatile boolean isUpgrading = false;
    private static volatile Thread upgradeThread = null;

    private static volatile int activeItemId = 0;
    private static volatile int activeItemType = 0;
    private static volatile String activeItemName = "";
    private static volatile int activeCurrentLevel = 0;
    private static volatile int activeTargetLevel = 0;

    private static volatile int clover3MinLevel = 0;
    private static volatile int clover3MaxLevel = 0;
    private static volatile int clover4MinLevel = 0;
    private static volatile int clover4MaxLevel = 0;
    private static volatile boolean stopWhenOutOfClover = true;
    private static volatile int actionDelayMs = 800;

    private static volatile int totalAttempts = 0;
    private static volatile int successCount = 0;
    private static volatile int failCount = 0;
    private static volatile String lastStatusMessage = "Chua kich hoat";

    // ANSI Color Constants for Terminal Output
    private static final String ANSI_RESET = "\u001B[0m";
    private static final String ANSI_RED = "\u001B[31m";
    private static final String ANSI_GREEN = "\u001B[32m";
    private static final String ANSI_YELLOW = "\u001B[33m";
    private static final String ANSI_CYAN = "\u001B[36m";
    private static final String ANSI_BOLD_GREEN = "\u001B[1;32m";
    private static final String ANSI_BOLD_RED = "\u001B[1;31m";
    private static final String ANSI_BOLD_YELLOW = "\u001B[1;33m";
    private static final String ANSI_BOLD_CYAN = "\u001B[1;36m";

    private static void resetServerResponseState() {
        try {
            ClassLoader cl = getGameClassLoader();
            Class<?> cClass = cl.loadClass("c");
            Field bField = cClass.getDeclaredField("B");
            Field fField = cClass.getDeclaredField("F");
            bField.setAccessible(true);
            fField.setAccessible(true);
            bField.setByte(null, (byte) 0);
            fField.set(null, null);
        } catch (Throwable ignored) {}
    }

    private static boolean isServerResponseArrived() {
        try {
            ClassLoader cl = getGameClassLoader();
            Class<?> cClass = cl.loadClass("c");
            Field bField = cClass.getDeclaredField("B");
            bField.setAccessible(true);
            return bField.getByte(null) == 1;
        } catch (Throwable ignored) {}
        return false;
    }

    private static String getServerResponseMsg() {
        try {
            ClassLoader cl = getGameClassLoader();
            Class<?> cClass = cl.loadClass("c");
            Field fField = cClass.getDeclaredField("F");
            fField.setAccessible(true);
            return (String) fField.get(null);
        } catch (Throwable ignored) {}
        return null;
    }

    private static void dismissPopupDialog() {
        try {
            ClassLoader cl = getGameClassLoader();
            Class<?> fuClass = cl.loadClass("fu");
            Field pField = fuClass.getDeclaredField("p");
            pField.setAccessible(true);
            Object pObj = pField.get(null);
            if (pObj != null) {
                Method fMethod = pObj.getClass().getDeclaredMethod("f");
                fMethod.setAccessible(true);
                fMethod.invoke(pObj);
            }
        } catch (Throwable ignored) {}
    }

    private static boolean isStopCondition(String msg) {
        if (msg == null || msg.trim().isEmpty()) return false;
        String lower = msg.toLowerCase();
        // Ignore standard upgrade failure or success dialogue
        if (lower.contains("cố gắng hết sức") || lower.contains("co gang het suc") ||
            lower.contains("thành công") || lower.contains("thanh cong") ||
            lower.contains("thất bại") || lower.contains("that bai")) {
            return false;
        }
        if (lower.contains("không đủ xu") || lower.contains("khong du xu") ||
            lower.contains("không đủ vàng") || lower.contains("khong du vang") ||
            lower.contains("không đủ tiền") || lower.contains("khong du tien") ||
            lower.contains("không đủ ngọc") || lower.contains("khong du ngoc") ||
            lower.contains("không đủ đá") || lower.contains("khong du da") ||
            lower.contains("thiếu đá") || lower.contains("thieu da") ||
            lower.contains("hành trang đầy") || lower.contains("hanh trang day") ||
            lower.contains("hành trang không đủ") || lower.contains("hanh trang khong du")) {
            return true;
        }
        return false;
    }

    public static class ItemInfo {
        public int id;
        public String name;
        public int type;
        public int level;
        public int quantity;

        public ItemInfo(int id, String name, int type, int level, int quantity) {
            this.id = id;
            this.name = name != null ? name : "";
            this.type = type;
            this.level = level;
            this.quantity = quantity;
        }
    }

    private static ClassLoader getGameClassLoader() {
        ClassLoader cl = Thread.currentThread().getContextClassLoader();
        if (cl != null) {
            return cl;
        }
        return AutoUpgrade.class.getClassLoader();
    }

    private static String escapeJson(String s) {
        if (s == null) return "";
        StringBuilder sb = new StringBuilder();
        for (int i = 0; i < s.length(); i++) {
            char c = s.charAt(i);
            switch (c) {
                case '"': sb.append("\\\""); break;
                case '\\': sb.append("\\\\"); break;
                case '\b': sb.append("\\b"); break;
                case '\f': sb.append("\\f"); break;
                case '\n': sb.append("\\n"); break;
                case '\r': sb.append("\\r"); break;
                case '\t': sb.append("\\t"); break;
                default:
                    if (c < ' ') {
                        String hex = Integer.toHexString(c);
                        sb.append("\\u");
                        for (int k = 0; k < 4 - hex.length(); k++) sb.append('0');
                        sb.append(hex);
                    } else {
                        sb.append(c);
                    }
                    break;
            }
        }
        return sb.toString();
    }

    private static boolean isClover3(String name) {
        if (name == null) return false;
        String lower = name.toLowerCase();
        return (lower.contains("cỏ 3") || lower.contains("co 3") ||
                lower.contains("3 lá") || lower.contains("3 la") ||
                lower.contains("cỏ ba lá") || lower.contains("co ba la") ||
                (lower.contains("cỏ") && lower.contains("3")) ||
                (lower.contains("co") && lower.contains("3")));
    }

    private static boolean isClover4(String name) {
        if (name == null) return false;
        String lower = name.toLowerCase();
        return (lower.contains("cỏ 4") || lower.contains("co 4") ||
                lower.contains("4 lá") || lower.contains("4 la") ||
                lower.contains("cỏ bốn lá") || lower.contains("co bon la") ||
                (lower.contains("cỏ") && lower.contains("4")) ||
                (lower.contains("co") && lower.contains("4")));
    }

    private static boolean isInsurance(String name) {
        if (name == null) return false;
        String lower = name.toLowerCase();
        return (lower.contains("bảo hiểm") || lower.contains("bao hiem"));
    }

    private static boolean isUpgradeStone(String name) {
        if (name == null) return false;
        String lower = name.toLowerCase();
        return (lower.contains("đá") || lower.contains("da") ||
                lower.contains("cường hóa") || lower.contains("cuong hoa") ||
                lower.contains("nâng cấp") || lower.contains("nang cap"));
    }

    /**
     * Reads current bag items from bw.V and player stats.
     */
    public static synchronized String getInventoryJson() {
        int gold = 0;
        int clover3Count = 0;
        int clover4Count = 0;
        int insuranceCount = 0;
        List<ItemInfo> equipmentItems = new ArrayList<ItemInfo>();

        try {
            ClassLoader cl = getGameClassLoader();
            Class<?> bwClass = cl.loadClass("bw");
            Field vField = bwClass.getDeclaredField("V");
            vField.setAccessible(true);
            Object vObj = vField.get(null);

            if (vObj != null) {
                Class<?> etClass = cl.loadClass("et");
                Method sizeMethod = etClass.getDeclaredMethod("c");
                Method getMethod = etClass.getDeclaredMethod("a", int.class);
                sizeMethod.setAccessible(true);
                getMethod.setAccessible(true);

                int size = ((Integer) sizeMethod.invoke(vObj)).intValue();

                Field idField = bwClass.getDeclaredField("O");
                Field typeField = bwClass.getDeclaredField("u");
                Field levelField = bwClass.getDeclaredField("z");
                Field nameField = bwClass.getDeclaredField("g");
                Field nameHField = bwClass.getDeclaredField("h");
                Field nameIField = bwClass.getDeclaredField("i");
                Field qtyField = bwClass.getDeclaredField("p");

                idField.setAccessible(true);
                typeField.setAccessible(true);
                levelField.setAccessible(true);
                nameField.setAccessible(true);
                nameHField.setAccessible(true);
                nameIField.setAccessible(true);
                qtyField.setAccessible(true);

                for (int i = 0; i < size; i++) {
                    Object itemObj = getMethod.invoke(vObj, i);
                    if (itemObj == null) continue;

                    int id = idField.getInt(itemObj);
                    int type = typeField.getInt(itemObj);
                    int level = levelField.getByte(itemObj) & 0xFF;
                    String name = (String) nameField.get(itemObj);
                    if (name == null || name.trim().isEmpty()) {
                        name = (String) nameHField.get(itemObj);
                    }
                    if (name == null || name.trim().isEmpty()) {
                        name = (String) nameIField.get(itemObj);
                    }
                    if (name == null) name = "Vật phẩm #" + id;

                    int qty = qtyField.getInt(itemObj);
                    if (qty <= 0) qty = 1;

                    // Material / Consumable count
                    if (type == 7 || isClover3(name) || isClover4(name) || isInsurance(name)) {
                        if (isClover3(name)) {
                            clover3Count += qty;
                        } else if (isClover4(name)) {
                            clover4Count += qty;
                        } else if (isInsurance(name)) {
                            insuranceCount += qty;
                        }
                    }

                    // Equipment items: category != 7
                    if (type != 7) {
                        equipmentItems.add(new ItemInfo(id, name, type, level, qty));
                    }
                }
            }

            // Read Gold from bq.g
            try {
                Class<?> bqClass = cl.loadClass("bq");
                Field goldField = bqClass.getDeclaredField("g");
                goldField.setAccessible(true);
                gold = goldField.getInt(null);
            } catch (Throwable t) {
                System.err.println("[AutoUpgrade] Note: bq.g not available: " + t.getMessage());
            }

        } catch (Throwable e) {
            System.err.println("[AutoUpgrade] Error reading inventory: " + e.getMessage());
            e.printStackTrace();
        }

        // Construct JSON response
        StringBuilder sb = new StringBuilder();
        sb.append("{");
        sb.append("\"gold\":").append(gold).append(",");
        sb.append("\"clover3Count\":").append(clover3Count).append(",");
        sb.append("\"clover4Count\":").append(clover4Count).append(",");
        sb.append("\"insuranceCount\":").append(insuranceCount).append(",");
        sb.append("\"items\":[");
        for (int i = 0; i < equipmentItems.size(); i++) {
            ItemInfo item = equipmentItems.get(i);
            if (i > 0) sb.append(",");
            sb.append("{");
            sb.append("\"id\":").append(item.id).append(",");
            sb.append("\"name\":\"").append(escapeJson(item.name)).append("\",");
            sb.append("\"type\":").append(item.type).append(",");
            sb.append("\"level\":").append(item.level).append(",");
            sb.append("\"quantity\":").append(item.quantity);
            sb.append("}");
        }
        sb.append("]");
        sb.append("}");

        return sb.toString();
    }

    private static Object findItemInInventory(int itemId, int itemType) {
        try {
            ClassLoader cl = getGameClassLoader();
            Class<?> bwClass = cl.loadClass("bw");
            Field vField = bwClass.getDeclaredField("V");
            vField.setAccessible(true);
            Object vObj = vField.get(null);
            if (vObj == null) return null;

            Class<?> etClass = cl.loadClass("et");
            Method sizeMethod = etClass.getDeclaredMethod("c");
            Method getMethod = etClass.getDeclaredMethod("a", int.class);
            sizeMethod.setAccessible(true);
            getMethod.setAccessible(true);

            int size = ((Integer) sizeMethod.invoke(vObj)).intValue();
            Field idField = bwClass.getDeclaredField("O");
            Field typeField = bwClass.getDeclaredField("u");
            idField.setAccessible(true);
            typeField.setAccessible(true);

            for (int i = 0; i < size; i++) {
                Object item = getMethod.invoke(vObj, i);
                if (item == null) continue;
                if (idField.getInt(item) == itemId && typeField.getInt(item) == itemType) {
                    return item;
                }
            }
        } catch (Throwable e) {
            System.err.println("[AutoUpgrade] findItemInInventory error: " + e.getMessage());
            e.printStackTrace();
        }
        return null;
    }

    private static Object findCloverInInventory(int cloverType) {
        // cloverType: 3 for Clover 3, 4 for Clover 4, 0 for Insurance
        try {
            ClassLoader cl = getGameClassLoader();
            Class<?> bwClass = cl.loadClass("bw");
            Field vField = bwClass.getDeclaredField("V");
            vField.setAccessible(true);
            Object vObj = vField.get(null);
            if (vObj == null) return null;

            Class<?> etClass = cl.loadClass("et");
            Method sizeMethod = etClass.getDeclaredMethod("c");
            Method getMethod = etClass.getDeclaredMethod("a", int.class);
            sizeMethod.setAccessible(true);
            getMethod.setAccessible(true);

            int size = ((Integer) sizeMethod.invoke(vObj)).intValue();
            Field nameField = bwClass.getDeclaredField("g");
            Field nameHField = bwClass.getDeclaredField("h");
            Field nameIField = bwClass.getDeclaredField("i");
            Field qtyField = bwClass.getDeclaredField("p");
            Field typeField = bwClass.getDeclaredField("u");
            Field aField = bwClass.getDeclaredField("A");

            nameField.setAccessible(true);
            nameHField.setAccessible(true);
            nameIField.setAccessible(true);
            qtyField.setAccessible(true);
            typeField.setAccessible(true);
            aField.setAccessible(true);

            for (int i = 0; i < size; i++) {
                Object item = getMethod.invoke(vObj, i);
                if (item == null) continue;
                String name = (String) nameField.get(item);
                if (name == null || name.trim().isEmpty()) {
                    name = (String) nameHField.get(item);
                }
                if (name == null || name.trim().isEmpty()) {
                    name = (String) nameIField.get(item);
                }
                int qty = qtyField.getInt(item);
                if (qty <= 0) qty = 1;
                int type = typeField.getInt(item);
                int aVal = aField.getByte(item) & 0xFF;

                if (cloverType == 3) {
                    if (isClover3(name) && qty > 0) return item;
                    if (type == 7 && aVal == 11 && qty > 0 && !isClover4(name)) return item;
                } else if (cloverType == 4) {
                    if (isClover4(name) && qty > 0) return item;
                } else if (cloverType == 0) {
                    if (isInsurance(name) && qty > 0) return item;
                }
            }
        } catch (Throwable e) {
            System.err.println("[AutoUpgrade] findCloverInInventory error: " + e.getMessage());
            e.printStackTrace();
        }
        return null;
    }

    private static int getItemLevel(Object itemObj) {
        if (itemObj == null) return 0;
        try {
            Class<?> bwClass = itemObj.getClass();
            Field levelField;
            try {
                levelField = bwClass.getDeclaredField("z");
            } catch (NoSuchFieldException nfe) {
                levelField = bwClass.getSuperclass().getDeclaredField("z");
            }
            levelField.setAccessible(true);
            return levelField.getByte(itemObj) & 0xFF;
        } catch (Throwable t) {
            System.err.println("[AutoUpgrade] getItemLevel error: " + t.getMessage());
            return 0;
        }
    }

    private static String getItemName(Object itemObj) {
        if (itemObj == null) return "";
        try {
            Class<?> bwClass = itemObj.getClass();
            Field nameField;
            try {
                nameField = bwClass.getDeclaredField("g");
            } catch (NoSuchFieldException nfe) {
                nameField = bwClass.getSuperclass().getDeclaredField("g");
            }
            nameField.setAccessible(true);
            String name = (String) nameField.get(itemObj);
            return name != null ? name : "";
        } catch (Throwable t) {
            return "";
        }
    }

    private static int getItemId(Object itemObj) {
        if (itemObj == null) return 0;
        try {
            Class<?> bwClass = itemObj.getClass();
            Field idField;
            try {
                idField = bwClass.getDeclaredField("O");
            } catch (NoSuchFieldException nfe) {
                idField = bwClass.getSuperclass().getDeclaredField("O");
            }
            idField.setAccessible(true);
            return idField.getInt(itemObj);
        } catch (Throwable t) {
            return 0;
        }
    }

    private static int getItemType(Object itemObj) {
        if (itemObj == null) return 0;
        try {
            Class<?> bwClass = itemObj.getClass();
            Field typeField;
            try {
                typeField = bwClass.getDeclaredField("u");
            } catch (NoSuchFieldException nfe) {
                typeField = bwClass.getSuperclass().getDeclaredField("u");
            }
            typeField.setAccessible(true);
            return typeField.getInt(itemObj);
        } catch (Throwable t) {
            return 0;
        }
    }

    /**
     * Dispatches packet q.a().b(b1, s2, b3)
     */
    private static void sendPacketB(byte b1, short s2, byte b3) {
        try {
            ClassLoader cl = getGameClassLoader();
            Class<?> qClass = cl.loadClass("q");
            Method getInstanceMethod = qClass.getDeclaredMethod("a");
            getInstanceMethod.setAccessible(true);
            Object qInstance = getInstanceMethod.invoke(null);
            if (qInstance == null) {
                System.err.println("[AutoUpgrade] q.a() returned null! Network service unavailable.");
                return;
            }

            Method bMethod = qClass.getDeclaredMethod("b", byte.class, short.class, byte.class);
            bMethod.setAccessible(true);
            bMethod.invoke(qInstance, b1, s2, b3);
        } catch (Throwable t) {
            System.err.println("[AutoUpgrade] Error sending packet b(" + b1 + ", " + s2 + ", " + b3 + "): " + t.getMessage());
            t.printStackTrace();
        }
    }

    /**
     * Launches background worker thread to execute automated upgrade loop.
     */
    public static synchronized void startUpgrade(
            int itemId,
            int itemType,
            int targetLevel,
            int clover3Min,
            int clover3Max,
            int clover4Min,
            int clover4Max,
            boolean stopOutOfClover,
            int delayMs
    ) {
        if (isUpgrading) {
            stopUpgrade();
        }

        activeItemId = itemId;
        activeItemType = itemType;
        activeTargetLevel = targetLevel;
        clover3MinLevel = clover3Min;
        clover3MaxLevel = clover3Max;
        clover4MinLevel = clover4Min;
        clover4MaxLevel = clover4Max;
        stopWhenOutOfClover = stopOutOfClover;
        actionDelayMs = Math.max(delayMs, 300);

        totalAttempts = 0;
        successCount = 0;
        failCount = 0;

        Object item = findItemInInventory(itemId, itemType);
        if (item == null) {
            lastStatusMessage = "Khong tim thay trang bi trong tui!";
            isUpgrading = false;
            return;
        }

        activeItemName = getItemName(item);
        activeCurrentLevel = getItemLevel(item);
        lastStatusMessage = "Bat dau dap " + activeItemName + " +" + activeCurrentLevel + " -> +" + activeTargetLevel;
        isUpgrading = true;

        upgradeThread = new Thread(new Runnable() {
            public void run() {
                System.out.println("[AutoUpgrade] Background upgrade worker started for item #" + activeItemId + " type=" + activeItemType + " (" + activeItemName + ") currentLv=" + activeCurrentLevel + " targetLv=" + activeTargetLevel);
                while (isUpgrading) {
                    try {
                        Object target = findItemInInventory(activeItemId, activeItemType);
                        if (target == null) {
                            lastStatusMessage = "Trang bi khong con trong tui do!";
                            isUpgrading = false;
                            break;
                        }

                        activeCurrentLevel = getItemLevel(target);
                        activeItemName = getItemName(target);

                        if (activeCurrentLevel >= activeTargetLevel) {
                            lastStatusMessage = "Chuc mung! Da dat +" + activeCurrentLevel + "!";
                            isUpgrading = false;
                            break;
                        }

                        // Determine lucky clover requirement
                        boolean needClover4 = (activeCurrentLevel >= clover4MinLevel && activeCurrentLevel <= clover4MaxLevel);
                        boolean needClover3 = (activeCurrentLevel >= clover3MinLevel && activeCurrentLevel <= clover3MaxLevel);

                        Object cloverToAttach = null;
                        if (needClover4) {
                            cloverToAttach = findCloverInInventory(4);
                            if (cloverToAttach == null && stopWhenOutOfClover) {
                                lastStatusMessage = "Het Co 4 la tai +" + activeCurrentLevel + "! Dung dap.";
                                isUpgrading = false;
                                break;
                            }
                        } else if (needClover3) {
                            cloverToAttach = findCloverInInventory(3);
                            if (cloverToAttach == null && stopWhenOutOfClover) {
                                lastStatusMessage = "Het Co 3 la tai +" + activeCurrentLevel + "! Dung dap.";
                                isUpgrading = false;
                                break;
                            }
                        }

                        int prevLevel = activeCurrentLevel;
                        totalAttempts++;
                        lastStatusMessage = "Dang dap +" + activeCurrentLevel + " -> +" + (activeCurrentLevel + 1) +
                                (cloverToAttach != null ? " (co co)" : "") + "...";

                        System.out.println(ANSI_CYAN + "[AutoUpgrade] " + ANSI_YELLOW + "Attempt #" + totalAttempts + ": " + ANSI_RESET +
                                "itemId=" + activeItemId + " type=" + activeItemType + " +" + prevLevel + " -> +" + (prevLevel + 1) +
                                (cloverToAttach != null ? " (co co)" : ""));

                        // Dismiss any open popup dialog before starting attempt
                        dismissPopupDialog();

                        // Reset server response tracker fields in class c
                        resetServerResponseState();

                        // Step 1: Put item into Blacksmith slot (Action 0)
                        sendPacketB((byte) 0, (short) activeItemId, (byte) activeItemType);
                        Thread.sleep(120);

                        // Step 2: Attach lucky clover if applicable (Action 0 for clover)
                        if (cloverToAttach != null) {
                            int cloverId = getItemId(cloverToAttach);
                            int cloverType = getItemType(cloverToAttach);
                            sendPacketB((byte) 0, (short) cloverId, (byte) cloverType);
                            Thread.sleep(120);
                        }

                        // Step 3: Select currency (Action 2: 0 = Vang/Xu)
                        sendPacketB((byte) 2, (short) 0, (byte) 0);
                        Thread.sleep(120);

                        // Step 4: Dispatch Blacksmith Upgrade Packet: q.a().b(5, (short)itemId, (byte)itemType)
                        sendPacketB((byte) 5, (short) activeItemId, (byte) activeItemType);

                        // Step 5: Wait for server response - poll level change and server message
                        long startWait = System.currentTimeMillis();
                        int pollDelay = Math.max(actionDelayMs, 1200);

                        while ((System.currentTimeMillis() - startWait < pollDelay) && isUpgrading) {
                            Thread.sleep(50);
                            String sMsg = getServerResponseMsg();
                            boolean respArrived = isServerResponseArrived();
                            Object itemCheck = findItemInInventory(activeItemId, activeItemType);
                            if (itemCheck != null) {
                                int checkLvl = getItemLevel(itemCheck);
                                if (checkLvl != prevLevel) {
                                    activeCurrentLevel = checkLvl;
                                    break;
                                }
                            }
                            if (respArrived || (sMsg != null && !sMsg.trim().isEmpty())) {
                                break;
                            }
                        }

                        // Inspect outcome
                        Object targetAfter = findItemInInventory(activeItemId, activeItemType);
                        if (targetAfter != null) {
                            activeCurrentLevel = getItemLevel(targetAfter);
                        }

                        String serverMsg = getServerResponseMsg();
                        if (serverMsg != null && !serverMsg.trim().isEmpty()) {
                            if (isStopCondition(serverMsg)) {
                                lastStatusMessage = serverMsg;
                                System.out.println(ANSI_BOLD_YELLOW + "[AutoUpgrade] Stopped due to server condition: " + serverMsg + ANSI_RESET);
                                isUpgrading = false;
                                break;
                            }
                        }

                        boolean hasResponse = isServerResponseArrived() || (serverMsg != null && !serverMsg.trim().isEmpty()) || (activeCurrentLevel != prevLevel);

                        if (!hasResponse) {
                            System.out.println(ANSI_YELLOW + "[AutoUpgrade] Attempt #" + totalAttempts + ": Server chua phan hoi trong thoi gian cho." + ANSI_RESET);
                            lastStatusMessage = "Dang cho server phan hoi...";
                        } else if (activeCurrentLevel > prevLevel) {
                            successCount++;
                            lastStatusMessage = "Thanh cong len +" + activeCurrentLevel + "! (" + successCount + "/" + totalAttempts + ")";
                            System.out.println(ANSI_BOLD_GREEN + "[AutoUpgrade] SUCCESS! +" + prevLevel + " -> +" + activeCurrentLevel + ANSI_RESET);
                        } else if (activeCurrentLevel < prevLevel) {
                            failCount++;
                            lastStatusMessage = "That bai tut xuong +" + activeCurrentLevel + "! (" + successCount + "/" + totalAttempts + ")";
                            System.out.println(ANSI_BOLD_RED + "[AutoUpgrade] FAILED! Downgraded to +" + activeCurrentLevel + ANSI_RESET);
                        } else {
                            failCount++;
                            if (serverMsg != null && !serverMsg.trim().isEmpty()) {
                                lastStatusMessage = serverMsg + " (+" + activeCurrentLevel + ") (" + successCount + "/" + totalAttempts + ")";
                            } else {
                                lastStatusMessage = "That bai giu nguyen +" + activeCurrentLevel + "! (" + successCount + "/" + totalAttempts + ")";
                            }
                            System.out.println(ANSI_RED + "[AutoUpgrade] FAILED! Kept +" + activeCurrentLevel + (serverMsg != null ? " [" + serverMsg + "]" : "") + ANSI_RESET);
                        }

                        // Dismiss any popup dialog so screen remains clean
                        dismissPopupDialog();

                        // Check if target reached
                        if (activeCurrentLevel >= activeTargetLevel) {
                            lastStatusMessage = "Chuc mung! Da dat +" + activeCurrentLevel + "!";
                            System.out.println(ANSI_BOLD_GREEN + "[AutoUpgrade] TARGET REACHED: +" + activeCurrentLevel + "! Congratulations!" + ANSI_RESET);
                            isUpgrading = false;
                            break;
                        }

                        // Rest delay between attempts
                        Thread.sleep(100);

                    } catch (InterruptedException ie) {
                        System.out.println(ANSI_YELLOW + "[AutoUpgrade] Background upgrade worker interrupted." + ANSI_RESET);
                        break;
                    } catch (Throwable err) {
                        System.err.println(ANSI_RED + "[AutoUpgrade] Error in upgrade loop: " + err.getMessage() + ANSI_RESET);
                        err.printStackTrace();
                        lastStatusMessage = "Loi nang cap: " + err.getMessage();
                        isUpgrading = false;
                        break;
                    }
                }
                isUpgrading = false;
                System.out.println(ANSI_CYAN + "[AutoUpgrade] Background upgrade worker stopped." + ANSI_RESET);
            }
        }, "AutoUpgrade-Worker");

        upgradeThread.setDaemon(true);
        upgradeThread.start();
    }

    /**
     * Halts active upgrade routine.
     */
    public static synchronized void stopUpgrade() {
        if (isUpgrading || (upgradeThread != null && upgradeThread.isAlive())) {
            isUpgrading = false;
            if (upgradeThread != null) {
                upgradeThread.interrupt();
                upgradeThread = null;
            }
            lastStatusMessage = "Da dung boi nguoi dung.";
            System.out.println("[AutoUpgrade] Upgrade halted by user.");
        }
    }

    /**
     * Returns JSON status string.
     */
    public static synchronized String getUpgradeStatusJson() {
        int clover3Count = 0;
        int clover4Count = 0;
        int insuranceCount = 0;

        // Query current material counts
        try {
            ClassLoader cl = getGameClassLoader();
            Class<?> bwClass = cl.loadClass("bw");
            Field vField = bwClass.getDeclaredField("V");
            vField.setAccessible(true);
            Object vObj = vField.get(null);

            if (vObj != null) {
                Class<?> etClass = cl.loadClass("et");
                Method sizeMethod = etClass.getDeclaredMethod("c");
                Method getMethod = etClass.getDeclaredMethod("a", int.class);
                sizeMethod.setAccessible(true);
                getMethod.setAccessible(true);

                int size = ((Integer) sizeMethod.invoke(vObj)).intValue();
                Field nameField = bwClass.getDeclaredField("g");
                Field qtyField = bwClass.getDeclaredField("p");
                nameField.setAccessible(true);
                qtyField.setAccessible(true);

                for (int i = 0; i < size; i++) {
                    Object item = getMethod.invoke(vObj, i);
                    if (item == null) continue;
                    String name = (String) nameField.get(item);
                    int qty = qtyField.getInt(item);
                    if (qty <= 0) qty = 1;

                    if (isClover3(name)) {
                        clover3Count += qty;
                    } else if (isClover4(name)) {
                        clover4Count += qty;
                    } else if (isInsurance(name)) {
                        insuranceCount += qty;
                    }
                }
            }
        } catch (Throwable t) {
            // Ignore material recount failure during status poll
        }

        StringBuilder sb = new StringBuilder();
        sb.append("{");
        sb.append("\"running\":").append(isUpgrading).append(",");
        sb.append("\"itemId\":").append(activeItemId).append(",");
        sb.append("\"itemName\":\"").append(escapeJson(activeItemName)).append("\",");
        sb.append("\"itemType\":").append(activeItemType).append(",");
        sb.append("\"currentLevel\":").append(activeCurrentLevel).append(",");
        sb.append("\"targetLevel\":").append(activeTargetLevel).append(",");
        sb.append("\"clover3Count\":").append(clover3Count).append(",");
        sb.append("\"clover4Count\":").append(clover4Count).append(",");
        sb.append("\"insuranceCount\":").append(insuranceCount).append(",");
        sb.append("\"attempts\":").append(totalAttempts).append(",");
        sb.append("\"successes\":").append(successCount).append(",");
        sb.append("\"failures\":").append(failCount).append(",");
        sb.append("\"message\":\"").append(escapeJson(lastStatusMessage)).append("\",");
        sb.append("\"stopped\":").append(!isUpgrading);
        sb.append("}");

        return sb.toString();
    }

    public static void main(String[] args) {
        System.out.println("=== Testing AutoUpgrade ===");
        System.out.println("Inventory JSON: " + getInventoryJson());
        System.out.println("Status JSON: " + getUpgradeStatusJson());
        startUpgrade(100, 0, 10, 4, 8, 9, 15, true, 500);
        System.out.println("Status JSON after start: " + getUpgradeStatusJson());
        stopUpgrade();
        System.out.println("Status JSON after stop: " + getUpgradeStatusJson());
        System.out.println("=== AutoUpgrade test complete ===");
    }
}
