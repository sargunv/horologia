import { Mail, User as UserIcon } from "lucide-react";
import { EditableField, PropertyRow } from "../EditableField.tsx";
import { SettingsSection } from "../space-settings/SettingsSection.tsx";

export function ProfileCard({
  userId,
  name,
  email,
}: {
  userId: string;
  name: string;
  email: string;
}) {
  return (
    <SettingsSection
      icon={<UserIcon className="size-5" aria-hidden="true" />}
      title="Profile"
      description="Your personal information."
    >
      <PropertyRow label="Name" icon={<UserIcon className="size-4" aria-hidden="true" />}>
        <EditableField userId={userId} field="name" value={name} label="Name" />
      </PropertyRow>
      <PropertyRow label="Email" icon={<Mail className="size-4" aria-hidden="true" />}>
        <EditableField userId={userId} field="email" value={email} label="Email" type="email" />
      </PropertyRow>
    </SettingsSection>
  );
}
