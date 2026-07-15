import { createFileRoute } from "@tanstack/react-router";
import { GlobalSearchCombobox } from "../../components/GlobalSearchCombobox.tsx";

export const Route = createFileRoute("/_authenticated/search")({
  component: SearchPage,
});

function SearchPage() {
  return (
    <div className="mx-auto max-w-3xl p-4 sm:p-6">
      <h1 className="mb-5 text-xl font-semibold">Search</h1>
      <GlobalSearchCombobox page />
    </div>
  );
}
