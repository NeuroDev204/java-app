package javax.microedition.rms;

public class RecordStore {
    private String name;
    private int recordStoreId;

    private RecordStore(String name, int recordStoreId) {
        this.name = name;
        this.recordStoreId = recordStoreId;
    }

    public String getName() {
        return name;
    }

    public static native RecordStore openRecordStore(String recordStoreName, boolean createIfNecessary) throws RecordStoreException;
    public native void closeRecordStore() throws RecordStoreException;
    public static native void deleteRecordStore(String recordStoreName) throws RecordStoreException;
    public static native String[] listRecordStores();

    public native int getNumRecords() throws RecordStoreException;
    public native byte[] getRecord(int recordId) throws RecordStoreException;
    public native int addRecord(byte[] saveData, int offset, int numBytes) throws RecordStoreException;
    public native void setRecord(int recordId, byte[] newData, int offset, int numBytes) throws RecordStoreException;
}
