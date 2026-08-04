// mirror จาก internal/bot/bot.go (ConfigField, ConfigFieldType) และ botSummary ใน
// internal/api/http/handler.go (response ของ GET /bots)
export type ConfigFieldType = "text" | "url";

export interface ConfigField {
  key: string;
  label: string;
  type: ConfigFieldType;
  required: boolean;
}

export interface Bot {
  name: string;
  config_schema: ConfigField[];
}
