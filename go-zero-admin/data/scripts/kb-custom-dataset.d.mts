/** 严格解码原始UTF-8字节，拒绝BOM与超过256KiB的输入。 */
export function decodeCustomDataset(bytes: ArrayBuffer | ArrayBufferView): string;
/** 返回已经按Go自定义v1规则验证的本地题集；不访问网络。 */
export function parseCustomDataset(raw: string): Promise<unknown>;
