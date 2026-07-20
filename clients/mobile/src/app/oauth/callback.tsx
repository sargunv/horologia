import { Redirect } from "expo-router";

import { routes } from "@/navigation/routes";

export default function OAuthCallbackScreen() {
  return <Redirect href={routes.root()} />;
}
